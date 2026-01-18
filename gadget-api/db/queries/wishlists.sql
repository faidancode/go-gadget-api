-- name: GetOrCreateWishlist :one
INSERT INTO wishlists (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
RETURNING *;

-- name: GetWishlistByUserID :one
SELECT * FROM wishlists WHERE user_id = $1 LIMIT 1;

-- name: AddWishlistItem :exec
INSERT INTO wishlist_items (wishlist_id, product_id)
VALUES ($1, $2)
ON CONFLICT (wishlist_id, product_id) DO NOTHING;

-- name: GetWishlistItems :many
SELECT wi.*, p.name, p.price, p.stock, p.image_url
FROM wishlist_items wi
JOIN products p ON wi.product_id = p.id
WHERE wi.wishlist_id = $1
ORDER BY wi.created_at DESC;

-- name: DeleteWishlistItem :exec
DELETE FROM wishlist_items
WHERE wishlist_id = $1 AND product_id = $2;

-- name: CheckWishlistItemExists :one
SELECT EXISTS(
    SELECT 1 FROM wishlist_items
    WHERE wishlist_id = $1 AND product_id = $2
) AS exists;