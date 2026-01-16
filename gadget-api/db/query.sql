-- name: ListCategories :many
SELECT *, count(*) OVER() AS total_count
FROM categories
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CreateCategory :one
INSERT INTO categories (name, slug, description, image_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories 
SET name = $2, slug = $3, description = $4, image_url = $5, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCategory :exec
UPDATE categories SET deleted_at = NOW() WHERE id = $1;

-- name: RestoreCategory :one
UPDATE categories SET deleted_at = NULL WHERE id = $1 RETURNING *;

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

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role VARCHAR(20) DEFAULT 'admin',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- name: CreateUser :one
INSERT INTO users (
    email,
    name,
    password,
    role
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, name, email, password, role, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password, role, created_at 
FROM users 
WHERE email = $1 
LIMIT 1;