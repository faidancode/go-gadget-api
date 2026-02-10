-- name: CreateReview :one
INSERT INTO reviews (user_id, product_id, order_id, rating, comment, is_verified_purchase)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetReviewByID :one
SELECT r.*, u.name as user_name, u.email as user_email
FROM reviews r
JOIN users u ON r.user_id = u.id
WHERE r.id = $1 AND r.deleted_at IS NULL
LIMIT 1;

-- name: GetReviewsByProductID :many
SELECT r.*, u.name as user_name
FROM reviews r
JOIN users u ON r.user_id = u.id
WHERE r.product_id = $1 AND r.deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetReviewsByUserID :many
SELECT r.*, p.name as product_name, p.slug as product_slug
FROM reviews r
JOIN products p ON r.product_id = p.id
WHERE r.user_id = $1 AND r.deleted_at IS NULL
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReviewsByProductID :one
SELECT COUNT(*) FROM reviews
WHERE product_id = $1 AND deleted_at IS NULL;

-- name: CountReviewsByUserID :one
SELECT COUNT(*) FROM reviews
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetAverageRatingByProductID :one
SELECT COALESCE(AVG(rating), 0) as average_rating
FROM reviews
WHERE product_id = $1 AND deleted_at IS NULL;

-- name: CheckReviewExists :one
SELECT EXISTS(
    SELECT 1 FROM reviews
    WHERE user_id = $1 AND product_id = $2 AND deleted_at IS NULL
) AS exists;

-- name: CheckUserPurchasedProduct :one
SELECT EXISTS(
    SELECT 1 FROM orders o
    JOIN order_items oi ON o.id = oi.order_id
    WHERE o.user_id = $1 
    AND oi.product_id = $2 
    AND o.status = 'COMPLETED'
    AND o.deleted_at IS NULL
) AS exists;

-- name: GetCompletedOrderForReview :one
SELECT o.id
FROM orders o
JOIN order_items oi ON o.id = oi.order_id
WHERE o.user_id = $1 
AND oi.product_id = $2 
AND o.status = 'COMPLETED'
AND o.deleted_at IS NULL
ORDER BY o.completed_at DESC
LIMIT 1;

-- name: UpdateReview :one
UPDATE reviews
SET rating = $2,
    comment = $3,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteReview :exec
UPDATE reviews
SET deleted_at = NOW()
WHERE id = $1;

-- name: GetUserRatingBreakdown :many
SELECT
  rating,
  COUNT(*) AS count
FROM reviews
WHERE user_id = $1
  AND deleted_at IS NULL
GROUP BY rating;
