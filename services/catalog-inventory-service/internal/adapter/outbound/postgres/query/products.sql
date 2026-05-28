-- name: ListProducts :many
SELECT
    p.id,
    p.category_id,
    c.name AS category_name,
    p.sku,
    p.name,
    COALESCE(p.brand, '')::text AS brand,
    p.unit,
    p.price,
    COALESCE(p.weight_kg, 0)::numeric AS weight_kg,
    p.requires_truck,
    (i.on_hand_qty - i.reserved_qty)::numeric AS available_qty
FROM products p
JOIN categories c ON c.id = p.category_id
JOIN inventories i ON i.product_id = p.id
WHERE p.is_active = true
  AND c.is_active = true
  AND (sqlc.arg(category_id)::text = '' OR p.category_id = sqlc.arg(category_id)::text)
  AND (sqlc.arg(search)::text = '' OR p.name ILIKE '%' || sqlc.arg(search)::text || '%' OR p.sku ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY p.name ASC
LIMIT sqlc.arg(limit_rows)::int
OFFSET sqlc.arg(offset_rows)::int;

-- name: CountProducts :one
SELECT count(*)::int
FROM products p
JOIN categories c ON c.id = p.category_id
WHERE p.is_active = true
  AND c.is_active = true
  AND (sqlc.arg(category_id)::text = '' OR p.category_id = sqlc.arg(category_id)::text)
  AND (sqlc.arg(search)::text = '' OR p.name ILIKE '%' || sqlc.arg(search)::text || '%' OR p.sku ILIKE '%' || sqlc.arg(search)::text || '%');

-- name: GetProduct :one
SELECT
    p.id,
    p.category_id,
    c.name AS category_name,
    p.sku,
    p.name,
    COALESCE(p.brand, '')::text AS brand,
    p.unit,
    p.price,
    COALESCE(p.weight_kg, 0)::numeric AS weight_kg,
    p.requires_truck,
    (i.on_hand_qty - i.reserved_qty)::numeric AS available_qty
FROM products p
JOIN categories c ON c.id = p.category_id
JOIN inventories i ON i.product_id = p.id
WHERE p.id = $1
  AND p.is_active = true
  AND c.is_active = true;

-- name: GetProductsByIDs :many
SELECT
    p.id,
    p.name,
    p.unit,
    p.price,
    (i.on_hand_qty - i.reserved_qty)::numeric AS available_qty
FROM products p
JOIN inventories i ON i.product_id = p.id
WHERE p.id = ANY($1::text[])
  AND p.is_active = true
ORDER BY p.id ASC;

