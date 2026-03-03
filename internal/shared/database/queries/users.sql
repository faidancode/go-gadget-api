-- name: CreateUser :one
INSERT INTO users (
    email,
    name,
    phone,
    password,
    role
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, name, email, phone, password, role, email_confirmed, created_at;

-- name: GetUserByEmail :one
SELECT id, email, name, phone, password, role, email_confirmed, created_at 
FROM users 
WHERE email = $1 
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, name, phone, password, role, email_confirmed, created_at 
FROM users 
WHERE id = $1 
LIMIT 1;

-- name: UpdateCustomerProfile :one
UPDATE users
SET
    name = $2,
    updated_at = NOW()
WHERE id = $1
  AND role = 'CUSTOMER'
RETURNING
    id,
    name,
    email,
    phone,
    role,
    email_confirmed,
    created_at,
    updated_at;

-- name: UpdateCustomerPassword :exec
UPDATE users
SET
    password = $2,
    updated_at = NOW()
WHERE id = $1
  AND role = 'CUSTOMER';

-- name: CheckPhoneExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE phone = $1
) AS exists;
