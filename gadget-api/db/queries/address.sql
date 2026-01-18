-- name: ListAddressesByUser :many
SELECT *, count(*) OVER() AS total_count
FROM addresses
WHERE user_id = $1
  AND deleted_at IS NULL
ORDER BY is_primary DESC, created_at DESC;

-- name: CreateAddress :one
INSERT INTO addresses (
    user_id, label, recipient_name, recipient_phone,
    street, subdistrict, district, city, province, postal_code, is_primary
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
)
RETURNING *;

-- name: UpdateAddress :one
UPDATE addresses
SET label = $2,
    recipient_name = $3,
    recipient_phone = $4,
    street = $5,
    subdistrict = $6,
    district = $7,
    city = $8,
    province = $9,
    postal_code = $10,
    is_primary = $11,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAddress :exec
UPDATE addresses
SET deleted_at = NOW()
WHERE id = $1
  AND user_id = $2
  AND deleted_at IS NULL;

-- name: UnsetPrimaryAddressByUser :exec
UPDATE addresses
SET is_primary = FALSE,
    updated_at = NOW()
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: ListAddressesAdmin :many
SELECT a.*, u.email, count(*) OVER() AS total_count
FROM addresses a
JOIN users u ON u.id = a.user_id
WHERE a.deleted_at IS NULL
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;

