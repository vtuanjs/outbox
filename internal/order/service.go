package order

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"outbox-demo/internal/outbox"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// PlaceOrder writes the order row AND the outbox event atomically.
//
// Write path:
//
//	BEGIN
//	  → INSERT INTO orders          (business data)
//	  → INSERT INTO outbox_events   (event row — same tx)
//	COMMIT
//
// Debezium reads the committed outbox row from the WAL and publishes to Kafka.
// If Kafka is down the row stays in WAL until Kafka recovers — no event loss.
func (s *Service) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*Order, error) {
	o := &Order{
		CustomerID: req.CustomerID,
		Amount:     req.Amount,
		Status:     "pending",
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after Commit

	// Step 1: insert business data
	if err := insertOrder(ctx, tx, o); err != nil {
		return nil, err
	}

	// Step 2: insert outbox event in the SAME transaction
	payload, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	if err := outbox.Insert(ctx, tx, outbox.Event{
		AggregateType: "order.placed", // → Kafka topic "order.placed"
		AggregateID:   o.ID,           // → Kafka partition key
		EventType:     "OrderPlaced",
		Payload:       payload,
	}); err != nil {
		return nil, err
	}

	// Step 3: single atomic commit — both rows land together or neither does
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return o, nil
}
