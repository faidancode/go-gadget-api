-- name: ListBrandsPublic :many
SELECT *, count(*) OVER() AS total_count
FROM brands
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBrandsAdmin :many
SELECT 
    *, 
    COUNT(*) OVER() AS total_count
FROM brands
WHERE 
    deleted_at IS NULL
    AND (
        sqlc.narg('search')::text IS NULL 
        OR name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR description ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY 
    -- Group 1: Sort berdasarkan STRING (Name, dll)
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'name' AND sqlc.arg('sort_dir')::text = 'asc' THEN name 
    END ASC,
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'name' AND sqlc.arg('sort_dir')::text = 'desc' THEN name 
    END DESC,

    -- Group 2: Sort berdasarkan TIMESTAMP (Created At)
    -- Kita pisahkan grup CASE agar tipe data tidak bentrok
    CASE 
        WHEN sqlc.arg('sort_col')::text = 'created_at' AND sqlc.arg('sort_dir')::text = 'asc' THEN created_at 
    END ASC,
    CASE 
        WHEN (sqlc.arg('sort_col')::text = 'created_at' AND sqlc.arg('sort_dir')::text = 'desc') 
             OR (sqlc.arg('sort_col')::text NOT IN ('name', 'created_at')) -- Fallback Logic di sini
             THEN created_at 
    END DESC
LIMIT $1 OFFSET $2;

-- name: GetBrandByID :one
SELECT * FROM brands WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CreateBrand :one
INSERT INTO brands (name, slug, description, image_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateBrand :one
UPDATE brands 
SET name = $2, slug = $3, description = $4, image_url = $5, is_active = $6, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteBrand :exec
UPDATE brands SET deleted_at = NOW() WHERE id = $1;

-- name: RestoreBrand :one
UPDATE brands SET deleted_at = NULL WHERE id = $1 RETURNING *;