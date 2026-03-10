package order

import "time"

type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Amount     int64     `json:"amount"` // cents
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type PlaceOrderRequest struct {
	CustomerID string `json:"customer_id"`
	Amount     int64  `json:"amount"` // cents
}
