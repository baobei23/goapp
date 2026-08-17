CREATE TABLE IF NOT EXISTS activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    entity_type VARCHAR(50),
    entity_id UUID,
    metadata JSONB,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX idx_activity_log_user_id ON activity_log(user_id);
CREATE INDEX idx_activity_log_event_type ON activity_log(event_type);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at DESC);
