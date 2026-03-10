package outbox

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID            string
	AggregateType string          // routes to Kafka topic (e.g. "order.placed")
	AggregateID   string          // Kafka partition key (e.g. order UUID)
	EventType     string          // e.g. "OrderPlaced"
	Payload       json.RawMessage
	CreatedAt     time.Time
}
