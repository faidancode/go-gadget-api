-- name: CreateOrder :one
INSERT INTO orders (
    order_number, user_id, status, address_snapshot, 
    subtotal_price, shipping_price, total_price, note, placed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
RETURNING *;

-- name: CreateOrderItem :exec
INSERT INTO order_items (
    order_id, product_id, name_snapshot, unit_price, quantity, total_price
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListOrders :many
SELECT o.*, count(*) OVER() AS total_count
FROM orders o
WHERE o.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;

-- name: ListOrdersAdmin :many
SELECT o.*, count(*) OVER() AS total_count
FROM orders o
WHERE (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL OR o.order_number ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1 LIMIT 1;

-- name: GetOrderItems :many
SELECT * FROM order_items WHERE order_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders 
SET status = $2, 
    updated_at = NOW(),
    completed_at = CASE WHEN $2 = 'COMPLETED' THEN NOW() ELSE completed_at END,
    cancelled_at = CASE WHEN $2 = 'CANCELLED' THEN NOW() ELSE cancelled_at END
WHERE id = $1
RETURNING *;