-- name: CreateOrder :one
INSERT INTO orders (
    order_number, user_id, status, address_id, address_snapshot, 
    subtotal_price, shipping_price, total_price, note, 
    snap_token, snap_redirect_url, snap_token_expired_at, placed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
RETURNING *;

-- name: CreateOrderItem :exec
INSERT INTO order_items (
    order_id, product_id, name_snapshot, unit_price, quantity, total_price
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListOrders :many
SELECT 
    o.id,
    o.order_number,
    o.status,
    o.total_price,
    o.placed_at,
    o.user_id,
    COUNT(*) OVER() AS total_count,
    (
        SELECT COALESCE(
            jsonb_agg(
                jsonb_build_object(
                    'id', oi.id,
                    'productId', oi.product_id,
                    'nameSnapshot', oi.name_snapshot,
                    'unitPrice', oi.total_price,
                    'quantity', oi.quantity,
                    'subtotal', oi.total_price * oi.quantity
                )
            ),
            '[]'::jsonb
        )
        FROM order_items oi
        WHERE oi.order_id = o.id
    )::jsonb AS items_json
FROM orders o
WHERE o.user_id = sqlc.arg('user_id')
  AND (
      sqlc.narg('status')::text IS NULL
      OR o.status = sqlc.narg('status')::text
  )
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;



-- name: ListOrdersAdmin :many
SELECT o.*, count(*) OVER() AS total_count
FROM orders o
WHERE o.deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL OR o.order_number ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrderByID :one
SELECT 
    o.id,
    o.order_number,
    o.user_id,
    o.status,
    o.payment_method,
    o.payment_status,
    o.address_snapshot,
    o.subtotal_price,
    o.discount_price,
    o.shipping_price,
    o.total_price,
    o.note,
    o.placed_at,
    o.paid_at,
    o.cancelled_at,
    o.cancel_reason,
    o.completed_at,
    o.receipt_no,
    o.snap_token,
    o.snap_redirect_url,
    o.snap_token_expired_at,
    (
        SELECT COALESCE(
            jsonb_agg(
                jsonb_build_object(
                    'id', oi.id,
                    'productId', oi.product_id,
                    'nameSnapshot', oi.name_snapshot,
                    'unitPrice', oi.unit_price,
                    'quantity', oi.quantity,
                    'subtotal', oi.total_price
                )
            ),
            '[]'::jsonb
        )
        FROM order_items oi
        WHERE oi.order_id = o.id
    )::jsonb AS items_json
FROM orders o
WHERE o.id = $1 
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: GetOrderSummaryByOrderNumber :one
SELECT
    id,
    order_number,
    subtotal_price,
    discount_price,
    shipping_price
FROM orders
WHERE order_number = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetOrderPaymentForUpdateByID :one
SELECT
    id,
    order_number,
    status,
    payment_status,
    payment_method,
    note,
    paid_at,
    cancelled_at
FROM orders
WHERE id = $1
  AND deleted_at IS NULL
FOR UPDATE;

-- name: GetOrderPaymentForUpdateByOrderNumber :one
SELECT
    id,
    order_number,
    status,
    payment_status,
    payment_method,
    note,
    paid_at,
    cancelled_at
FROM orders
WHERE order_number = $1
  AND deleted_at IS NULL
FOR UPDATE;

-- name: GetOrderItems :many
SELECT 
    id, 
    order_id, 
    product_id, 
    name_snapshot, 
    unit_price, 
    quantity, 
    total_price
FROM order_items 
WHERE order_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders 
SET status = @status::text, 
    updated_at = NOW(),
    completed_at = CASE WHEN @status::text = 'COMPLETED' THEN NOW() ELSE completed_at END,
    cancelled_at = CASE WHEN @status::text = 'CANCELLED' THEN NOW() ELSE cancelled_at END
WHERE id = $1
RETURNING *;

-- name: UpdateOrderPaymentStatus :one
UPDATE orders
SET
    payment_status = @payment_status::text,
    payment_method = CASE WHEN @payment_method::text IS NULL THEN payment_method ELSE @payment_method::text END,
    paid_at = @paid_at,
    cancelled_at = @cancelled_at,
    status = @status::text,
    note = CASE WHEN @note::text IS NULL THEN note ELSE @note::text END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOrderSnapToken :one
UPDATE orders
SET
    snap_token = $2,
    snap_redirect_url = $3,
    snap_token_expired_at = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
