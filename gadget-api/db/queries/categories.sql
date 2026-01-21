-- name: ListCategoriesPublic :many
SELECT *, count(*) OVER() AS total_count
FROM categories
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListCategoriesAdmin :many
SELECT 
    *, 
    COUNT(*) OVER() AS total_count
FROM categories
WHERE 
    deleted_at IS NULL
    AND (
        sqlc.narg('search')::text IS NULL 
        OR name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR description ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY 
    -- Sort by Name
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'name' AND sqlc.arg('sort_dir')::text = 'asc' 
            THEN name 
    END ASC,
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'name' AND sqlc.arg('sort_dir')::text = 'desc' 
            THEN name 
    END DESC,
    -- Sort by CreatedAt (Default)
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'created_at' AND sqlc.arg('sort_dir')::text = 'asc' 
            THEN created_at 
    END ASC,
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'created_at' AND sqlc.arg('sort_dir')::text = 'desc' 
            THEN created_at 
    END DESC,
    -- Fallback jika tidak ada sort yang cocok
    created_at DESC
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