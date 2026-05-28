-- name: CreateOrder :exec
INSERT INTO orders (id, customer_name, customer_phone, customer_address, status, total_amount, payment_id, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: CreateOrderItem :exec
INSERT INTO order_items (id, order_id, product_id, product_name, unit, quantity, unit_price, line_total)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetOrder :one
SELECT id, customer_name, customer_phone, customer_address, status, total_amount, payment_id, correlation_id
       , created_at, updated_at
FROM orders
WHERE id = $1;

-- name: ListOrderItems :many
SELECT id, order_id, product_id, product_name, unit, quantity, unit_price, line_total
FROM order_items
WHERE order_id = $1
ORDER BY id;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateOrderStatusWithPayment :exec
UPDATE orders
SET status = $2, payment_id = $3, updated_at = now()
WHERE id = $1;

-- name: CreateSagaInstance :exec
INSERT INTO saga_instances (id, order_id, status, current_step, correlation_id)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateSagaInstance :exec
UPDATE saga_instances
SET status = $2,
    current_step = $3,
    completed_at = CASE WHEN $4::boolean THEN now() ELSE completed_at END
WHERE order_id = $1;

-- name: CreateSagaStep :exec
INSERT INTO saga_steps (id, saga_id, step_name, status, idempotency_key, error_message)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (id, aggregate_id, aggregate_type, event_type, correlation_id, causation_id, traceparent, payload, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
