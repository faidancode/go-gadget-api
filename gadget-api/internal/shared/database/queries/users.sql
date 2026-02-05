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
SELECT id, email, name, password, role, created_at 
FROM users 
WHERE email = $1 
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, name, password, role, created_at 
FROM users 
WHERE id = $1 
LIMIT 1;