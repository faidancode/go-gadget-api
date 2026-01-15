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

-- name: ListProducts :many
SELECT p.*, c.name as category_name, count(*) OVER() AS total_count
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('search')::text IS NULL OR p.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('category_id')::uuid IS NULL OR p.category_id = sqlc.narg('category_id')::uuid)
ORDER BY p.created_at DESC
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
SET category_id = $2, name = $3, slug = $4, description = $5, price = $6, stock = $7, sku = $8, image_url = $9, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET deleted_at = NOW() WHERE id = $1;

-- name: RestoreProduct :one
UPDATE products SET deleted_at = NULL WHERE id = $1 RETURNING *;