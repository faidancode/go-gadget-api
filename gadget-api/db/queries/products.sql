
-- name: ListProductsPublic :many
SELECT p.*, c.name as category_name, count(*) OVER() AS total_count
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.deleted_at IS NULL 
  AND p.is_active = true
  -- Gunakan sintaks ini agar sqlc membuat field CategoryID (NullUUID)
  AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR p.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (p.price >= sqlc.arg('min_price')::decimal)
  AND (p.price <= sqlc.arg('max_price')::decimal)
ORDER BY 
    CASE WHEN sqlc.arg('sort_by')::text = 'newest' THEN p.created_at END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'oldest' THEN p.created_at END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'price_high' THEN p.price END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'price_low' THEN p.price END ASC,
    p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListProductsAdmin :many
SELECT p.*, c.name as category_name, count(*) OVER() AS total_count
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR p.name ILIKE '%' || sqlc.narg('search')::text || '%' OR p.sku ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY 
    CASE WHEN sqlc.arg('sort_col')::text = 'stock' THEN p.stock END ASC,
    CASE WHEN sqlc.arg('sort_col')::text = 'name' THEN p.name END ASC,
    p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetProductByID :one
SELECT p.*, c.name as category_name 
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.id = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: CreateProduct :one
INSERT INTO products (category_id, name, slug, description, price, stock, sku, image_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET 
    category_id = $2,
    name = $3,
    description = $4,
    price = $5,
    stock = $6,
    sku = $7,
    image_url = $8,
    is_active = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET deleted_at = NOW() WHERE id = $1;

-- name: RestoreProduct :one
UPDATE products SET deleted_at = NULL WHERE id = $1 RETURNING *;