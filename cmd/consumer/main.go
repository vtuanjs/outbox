package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"

	"outbox-demo/internal/db"
)

// eventPayload mirrors the fields we need from the order payload.
type eventPayload struct {
	ID string `json:"id"`
}

// debeziumEnvelope is the wrapper Debezium adds when VALUE_CONVERTER is JsonConverter.
// The payload field may be a JSON string (JSONB without expand) or a JSON object
// (JSONB with expand.json.payload=true), so we use RawMessage to handle both.
type debeziumEnvelope struct {
	Payload json.RawMessage `json:"payload"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "order.placed"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: "order-consumer",
	})
	defer r.Close()

	log.Printf("consuming topic %q from %s", topic, broker)

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break // clean shutdown
			}
			log.Printf("fetch error: %v", err)
			continue
		}

		if err := handle(ctx, pool, msg); err != nil {
			log.Printf("handle error: %v", err)
			// do not commit — will be redelivered on next poll
			continue
		}

		// Commit only after successful DB update — at-least-once guarantee
		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("commit error: %v", err)
		}
	}

	log.Println("consumer stopped")
}

// handle parses the event and updates the order status to "consumed".
// Idempotent: UPDATE WHERE status != 'consumed' is a no-op on replay.
func handle(ctx context.Context, pool *pgxpool.Pool, msg kafka.Message) error {
	// Debezium wraps the message in {"schema":...,"payload":...}.
	// payload may be a JSON string (raw JSONB) or a JSON object (expand.json.payload=true).
	raw := msg.Value
	var envelope debeziumEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err == nil && len(envelope.Payload) > 0 {
		var inner string
		if err := json.Unmarshal(envelope.Payload, &inner); err == nil {
			raw = []byte(inner) // payload was a quoted JSON string
		} else {
			raw = envelope.Payload // payload is already a JSON object
		}
	}

	var p eventPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	if p.ID == "" {
		return fmt.Errorf("missing id in payload: %s", msg.Value)
	}

	tag, err := pool.Exec(ctx,
		`UPDATE orders SET status = 'consumed' WHERE id = $1 AND status != 'consumed'`,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("update order %s: %w", p.ID, err)
	}

	log.Printf("order %s → consumed (rows affected: %d)", p.ID, tag.RowsAffected())
	return nil
}
