.PHONY: up down deps run run-consumer register-connector connector-status connector-delete connector-recover

# ── Infrastructure ─────────────────────────────────────────────────────────────

up:
	docker compose up -d

down:
	docker compose down -v

# ── Go ─────────────────────────────────────────────────────────────────────────

deps:
	go mod tidy

run:
	go run ./cmd/server

run-consumer:
	go run ./cmd/consumer

# ── Debezium connector ─────────────────────────────────────────────────────────

# Register the outbox connector once Kafka Connect is healthy
register-connector:
	@echo "Waiting for Kafka Connect..."
	@until curl -sf http://localhost:8083/connectors > /dev/null; do sleep 3; done
	curl -X POST http://localhost:8083/connectors \
		-H "Content-Type: application/json" \
		-d @debezium/connector.json
	@echo ""

connector-status:
	curl -s http://localhost:8083/connectors/outbox-connector/status | python3 -m json.tool

connector-delete:
	curl -X DELETE http://localhost:8083/connectors/outbox-connector

# Recovery after replication slot invalidation (e.g. WAL disk cap hit).
# snapshot.mode=initial only snapshots when there is no stored offset.
# Deleting + re-registering the connector clears the stored offset, which
# triggers a fresh snapshot on restart — equivalent to "when_needed" behaviour.
connector-recover:
	@echo "Step 1: delete connector (clears stored offset)..."
	-curl -sf -X DELETE http://localhost:8083/connectors/outbox-connector
	@sleep 2
	@echo "Step 2: drop invalidated replication slot (if still registered in postgres)..."
	-docker compose exec -T postgres psql -U user -d mydb \
		-c "SELECT pg_drop_replication_slot('debezium_outbox_slot') WHERE EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = 'debezium_outbox_slot');"
	@sleep 1
	@echo "Step 3: re-register connector (no stored offset → fresh snapshot)..."
	curl -X POST http://localhost:8083/connectors \
		-H "Content-Type: application/json" \
		-d @debezium/connector.json
	@echo ""

# ── Test ───────────────────────────────────────────────────────────────────────

# Send a sample PlaceOrder request
place-order:
	curl -s -X POST http://localhost:8080/orders \
		-H "Content-Type: application/json" \
		-d '{"customer_id":"cust-123","amount":4999}' | python3 -m json.tool

# Watch outbox_events accumulate (useful for verifying writes before Debezium is up)
watch-outbox:
	docker compose exec postgres psql -U user -d mydb \
		-c "SELECT id, aggregate_type, event_type, created_at FROM outbox_events ORDER BY created_at DESC LIMIT 20;"
