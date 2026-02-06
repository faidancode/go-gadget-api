-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (
  id, aggregate_type, aggregate_id, event_type, payload, status
) VALUES ($1, $2, $3, $4, $5, 'PENDING');

-- name: ListPendingOutbox :many
SELECT * FROM outbox_events
WHERE status = 'PENDING'
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxSent :exec
UPDATE outbox_events
SET status = 'SENT', processed_at = NOW()
WHERE id = $1;
