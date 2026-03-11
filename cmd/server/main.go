package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"outbox-demo/internal/db"
	"outbox-demo/internal/order"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	svc := order.NewService(pool)

	mux := http.NewServeMux()

	// GET /health — liveness probe
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// POST /orders — place an order (demonstrates the outbox pattern)
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req order.PlaceOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		o, err := svc.PlaceOrder(r.Context(), req)
		if err != nil {
			log.Printf("PlaceOrder error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(o)
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
