-- name: GetReservationByIdempotencyKey :one
SELECT id, order_id, status, idempotency_key
FROM stock_reservations
WHERE idempotency_key = $1;

-- name: GetReservationByOrderID :one
SELECT id, order_id, status, idempotency_key
FROM stock_reservations
WHERE order_id = $1;

-- name: ListReservationItems :many
SELECT product_id, quantity
FROM stock_reservation_items
WHERE reservation_id = $1
ORDER BY product_id ASC;

-- name: LockInventoriesForProducts :many
SELECT product_id, on_hand_qty, reserved_qty
FROM inventories
WHERE product_id = ANY($1::text[])
FOR UPDATE;

-- name: CreateReservation :one
INSERT INTO stock_reservations (id, order_id, status, idempotency_key)
VALUES ($1, $2, $3, $4)
RETURNING id, order_id, status, idempotency_key;

-- name: CreateReservationItem :exec
INSERT INTO stock_reservation_items (id, reservation_id, product_id, quantity)
VALUES ($1, $2, $3, $4);

-- name: AddReservedStock :exec
UPDATE inventories
SET reserved_qty = reserved_qty + sqlc.arg(quantity)::numeric,
    updated_at = now()
WHERE product_id = sqlc.arg(product_id)::text;

-- name: ReleaseReservedStock :exec
UPDATE inventories
SET reserved_qty = reserved_qty - sqlc.arg(quantity)::numeric,
    updated_at = now()
WHERE product_id = sqlc.arg(product_id)::text;

-- name: CommitReservedStock :exec
UPDATE inventories
SET on_hand_qty = on_hand_qty - sqlc.arg(quantity)::numeric,
    reserved_qty = reserved_qty - sqlc.arg(quantity)::numeric,
    updated_at = now()
WHERE product_id = sqlc.arg(product_id)::text;

-- name: UpdateReservationStatus :one
UPDATE stock_reservations
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, order_id, status, idempotency_key;

