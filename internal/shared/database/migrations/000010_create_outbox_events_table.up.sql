CREATE TABLE
    outbox_events (
        id UUID PRIMARY KEY,
        aggregate_type VARCHAR(50) NOT NULL,
        aggregate_id UUID NOT NULL,
        event_type VARCHAR(50) NOT NULL,
        payload JSONB NOT NULL,
        status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        processed_at TIMESTAMPTZ
    );

CREATE INDEX idx_outbox_events_status_created_at ON outbox_events (status, created_at);