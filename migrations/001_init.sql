-- Business table
CREATE TABLE IF NOT EXISTS orders (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id VARCHAR(255) NOT NULL,
    amount      BIGINT       NOT NULL, -- in cents
    status      VARCHAR(50)  NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Outbox table (Debezium CDC contract — do not rename columns)
CREATE TABLE IF NOT EXISTS outbox_events (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type  VARCHAR(255) NOT NULL,  -- routes to Kafka topic
    aggregate_id    VARCHAR(255) NOT NULL,  -- Kafka partition key
    event_type      VARCHAR(255) NOT NULL,  -- e.g. "OrderPlaced"
    payload         JSONB        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
