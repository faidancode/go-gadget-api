package auth

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=auth_repo.go -destination=../mock/auth/auth_repo_mock.go -package=mock
type Repository interface {
	Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error)
	GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error)
	GetUserProfileByEmail(ctx context.Context, email string) (UserProfile, error)
	GetLatestPasswordResetTokenByUserID(ctx context.Context, userID uuid.UUID) (PasswordResetTokenRecord, error)
	UpsertPasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt, createdAt time.Time) error
	GetPasswordResetToken(ctx context.Context, token string) (PasswordResetTokenRecord, error)
	DeletePasswordResetTokenByToken(ctx context.Context, token string) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error
	GetLatestEmailConfirmationTokenByUserID(ctx context.Context, userID uuid.UUID) (EmailConfirmationTokenRecord, error)
	UpsertEmailConfirmationToken(ctx context.Context, userID uuid.UUID, token, pin string, expiresAt, createdAt time.Time) error
	DeleteEmailConfirmationTokensByUserID(ctx context.Context, userID uuid.UUID) error
	GetEmailConfirmationTokenByToken(ctx context.Context, token string) (EmailConfirmationTokenRecord, error)
	DeleteEmailConfirmationTokenByToken(ctx context.Context, token string) error
	DeleteEmailConfirmationTokenByPin(ctx context.Context, pin string) error
	SetUserEmailConfirmed(ctx context.Context, userID uuid.UUID) error
}

type UserProfile struct {
	ID             uuid.UUID
	Email          string
	Name           string
	Role           string
	EmailConfirmed bool
}

type PasswordResetTokenRecord struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type EmailConfirmationTokenRecord struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	Pin       string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type repository struct {
	queries *dbgen.Queries
	db      *sql.DB
}

func NewRepository(q *dbgen.Queries, db *sql.DB) Repository {
	return &repository{
		queries: q,
		db:      db,
	}
}

func (r *repository) GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *repository) Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error) {
	return r.queries.CreateUser(ctx, params)
}

func (r *repository) GetUserProfileByEmail(ctx context.Context, email string) (UserProfile, error) {
	const query = `
		SELECT id, email, name, role, email_confirmed
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var out UserProfile
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&out.ID,
		&out.Email,
		&out.Name,
		&out.Role,
		&out.EmailConfirmed,
	)
	return out, err
}

func (r *repository) GetLatestPasswordResetTokenByUserID(ctx context.Context, userID uuid.UUID) (PasswordResetTokenRecord, error) {
	const query = `
		SELECT id, user_id, token, created_at, expires_at
		FROM password_reset_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var out PasswordResetTokenRecord
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&out.ID,
		&out.UserID,
		&out.Token,
		&out.CreatedAt,
		&out.ExpiresAt,
	)
	return out, err
}

func (r *repository) UpsertPasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt, createdAt time.Time) error {
	const query = `
		INSERT INTO password_reset_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at
	`
	_, err := r.db.ExecContext(ctx, query, userID, token, expiresAt, createdAt)
	return err
}

func (r *repository) GetPasswordResetToken(ctx context.Context, token string) (PasswordResetTokenRecord, error) {
	const query = `
		SELECT id, user_id, token, created_at, expires_at
		FROM password_reset_tokens
		WHERE token = $1
		LIMIT 1
	`

	var out PasswordResetTokenRecord
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&out.ID,
		&out.UserID,
		&out.Token,
		&out.CreatedAt,
		&out.ExpiresAt,
	)
	return out, err
}

func (r *repository) DeletePasswordResetTokenByToken(ctx context.Context, token string) error {
	const query = `DELETE FROM password_reset_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *repository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	return r.queries.UpdateCustomerPassword(ctx, dbgen.UpdateCustomerPasswordParams{
		ID:       userID,
		Password: password,
	})
}

func (r *repository) GetLatestEmailConfirmationTokenByUserID(ctx context.Context, userID uuid.UUID) (EmailConfirmationTokenRecord, error) {
	const query = `
		SELECT id, user_id, token, pin, created_at, expires_at
		FROM email_confirmation_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var out EmailConfirmationTokenRecord
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&out.ID,
		&out.UserID,
		&out.Token,
		&out.Pin,
		&out.CreatedAt,
		&out.ExpiresAt,
	)
	return out, err
}

func (r *repository) UpsertEmailConfirmationToken(ctx context.Context, userID uuid.UUID, token, pin string, expiresAt, createdAt time.Time) error {
	const query = `
		INSERT INTO email_confirmation_tokens (user_id, token, pin, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id)
		DO UPDATE SET token = EXCLUDED.token, pin = EXCLUDED.pin, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at
	`
	_, err := r.db.ExecContext(ctx, query, userID, token, pin, expiresAt, createdAt)
	return err
}

func (r *repository) DeleteEmailConfirmationTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `DELETE FROM email_confirmation_tokens WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *repository) GetEmailConfirmationTokenByToken(ctx context.Context, token string) (EmailConfirmationTokenRecord, error) {
	const query = `
		SELECT id, user_id, token, pin, created_at, expires_at
		FROM email_confirmation_tokens
		WHERE token = $1
		LIMIT 1
	`

	var out EmailConfirmationTokenRecord
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&out.ID,
		&out.UserID,
		&out.Token,
		&out.Pin,
		&out.CreatedAt,
		&out.ExpiresAt,
	)
	return out, err
}

func (r *repository) DeleteEmailConfirmationTokenByToken(ctx context.Context, token string) error {
	const query = `DELETE FROM email_confirmation_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *repository) DeleteEmailConfirmationTokenByPin(ctx context.Context, pin string) error {
	const query = `DELETE FROM email_confirmation_tokens WHERE pin = $1`
	_, err := r.db.ExecContext(ctx, query, pin)
	return err
}

func (r *repository) SetUserEmailConfirmed(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE users
		SET email_confirmed = true, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
