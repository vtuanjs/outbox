package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Insert writes an outbox event within an existing transaction.
// Must be called in the same tx as the business write — that's the whole point.
func Insert(ctx context.Context, tx pgx.Tx, e Event) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`, e.AggregateType, e.AggregateID, e.EventType, string(e.Payload))
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}
