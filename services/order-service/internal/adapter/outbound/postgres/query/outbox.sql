-- name: ListPendingOutboxEvents :many
SELECT id, aggregate_id, aggregate_type, event_type, correlation_id, causation_id, traceparent, payload, status, created_at
FROM outbox_events
WHERE status = 'PENDING'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'PUBLISHED',
    published_at = now()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET retry_count = retry_count + 1
WHERE id = $1;
