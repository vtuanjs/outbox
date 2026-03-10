package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func insertOrder(ctx context.Context, tx pgx.Tx, o *Order) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO orders (customer_id, amount, status)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at
	`, o.CustomerID, o.Amount, o.Status).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}
