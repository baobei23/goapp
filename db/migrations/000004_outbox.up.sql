CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,
    entity_id UUID NOT NULL,
    metadata JSONB,
    created_at timestamptz DEFAULT now(),
    processed_at timestamptz,
    retries INT DEFAULT 0
);

CREATE INDEX idx_outbox_pending ON outbox(processed_at) WHERE processed_at IS NULL;
CREATE INDEX idx_outbox_created_at ON outbox(created_at);
