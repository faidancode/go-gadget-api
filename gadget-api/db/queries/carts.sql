-- name: CreateCart :one
INSERT INTO carts (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: GetCartByUserID :one
SELECT *
FROM carts
WHERE user_id = $1
LIMIT 1;

-- name: CountCartItems :one
SELECT COALESCE(SUM(quantity), 0)::bigint
FROM cart_items
WHERE cart_id = $1;

-- name: AddCartItem :exec
INSERT INTO cart_items (cart_id, product_id, quantity, price_at_add)
VALUES ($1, $2, $3, $4)
ON CONFLICT (cart_id, product_id)
DO UPDATE SET
  quantity = cart_items.quantity + EXCLUDED.quantity,
  updated_at = NOW();

-- name: UpdateCartItemQty :one
UPDATE cart_items
SET quantity = $3, updated_at = NOW()
WHERE cart_id = $1 AND product_id = $2
RETURNING *;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: DeleteCart :exec
DELETE FROM carts
WHERE id = $1;

-- name: GetCartDetail :many
SELECT
  ci.id,
  ci.product_id,
  ci.quantity,
  ci.price_at_add,
  ci.created_at
FROM carts c
JOIN cart_items ci ON ci.cart_id = c.id
WHERE c.user_id = $1
ORDER BY ci.created_at DESC;
