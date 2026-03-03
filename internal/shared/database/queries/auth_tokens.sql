-- name: GetLatestPasswordResetTokenByUserID :one
SELECT id, user_id, token, created_at, expires_at
FROM password_reset_tokens
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: UpsertPasswordResetToken :exec
INSERT INTO password_reset_tokens (user_id, token, expires_at, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id)
DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at;

-- name: GetPasswordResetToken :one
SELECT id, user_id, token, created_at, expires_at
FROM password_reset_tokens
WHERE token = $1
LIMIT 1;

-- name: DeletePasswordResetTokenByToken :exec
DELETE FROM password_reset_tokens WHERE token = $1;

-- name: GetLatestEmailConfirmationTokenByUserID :one
SELECT id, user_id, token, pin, created_at, expires_at
FROM email_confirmation_tokens
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: UpsertEmailConfirmationToken :exec
INSERT INTO email_confirmation_tokens (user_id, token, pin, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id)
DO UPDATE SET token = EXCLUDED.token, pin = EXCLUDED.pin, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at;

-- name: DeleteEmailConfirmationTokensByUserID :exec
DELETE FROM email_confirmation_tokens WHERE user_id = $1;

-- name: GetEmailConfirmationTokenByToken :one
SELECT id, user_id, token, pin, created_at, expires_at
FROM email_confirmation_tokens
WHERE token = $1
LIMIT 1;

-- name: DeleteEmailConfirmationTokenByToken :exec
DELETE FROM email_confirmation_tokens WHERE token = $1;

-- name: DeleteEmailConfirmationTokenByPin :exec
DELETE FROM email_confirmation_tokens WHERE pin = $1;

-- name: SetUserEmailConfirmed :exec
UPDATE users
SET email_confirmed = true, updated_at = NOW()
WHERE id = $1;
