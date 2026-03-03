package auth

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=auth_repo.go -destination=../mock/auth/auth_repo_mock.go -package=mock
type Repository interface {
	Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error)
	GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error)
	CheckPhoneExists(ctx context.Context, phone sql.NullString) (bool, error)
	GetLatestPasswordResetTokenByUserID(ctx context.Context, userID uuid.UUID) (dbgen.PasswordResetToken, error)
	UpsertPasswordResetToken(ctx context.Context, params dbgen.UpsertPasswordResetTokenParams) error
	GetPasswordResetToken(ctx context.Context, token string) (dbgen.PasswordResetToken, error)
	DeletePasswordResetTokenByToken(ctx context.Context, token string) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error
	GetLatestEmailConfirmationTokenByUserID(ctx context.Context, userID uuid.UUID) (dbgen.EmailConfirmationToken, error)
	UpsertEmailConfirmationToken(ctx context.Context, params dbgen.UpsertEmailConfirmationTokenParams) error
	DeleteEmailConfirmationTokensByUserID(ctx context.Context, userID uuid.UUID) error
	GetEmailConfirmationTokenByToken(ctx context.Context, token string) (dbgen.EmailConfirmationToken, error)
	DeleteEmailConfirmationTokenByToken(ctx context.Context, token string) error
	DeleteEmailConfirmationTokenByPin(ctx context.Context, pin string) error
	SetUserEmailConfirmed(ctx context.Context, userID uuid.UUID) error
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

func (r *repository) CheckPhoneExists(ctx context.Context, phone sql.NullString) (bool, error) {
	return r.queries.CheckPhoneExists(ctx, phone)
}

func (r *repository) GetLatestPasswordResetTokenByUserID(ctx context.Context, userID uuid.UUID) (dbgen.PasswordResetToken, error) {
	return r.queries.GetLatestPasswordResetTokenByUserID(ctx, userID)
}

func (r *repository) UpsertPasswordResetToken(ctx context.Context, params dbgen.UpsertPasswordResetTokenParams) error {
	return r.queries.UpsertPasswordResetToken(ctx, params)
}

func (r *repository) GetPasswordResetToken(ctx context.Context, token string) (dbgen.PasswordResetToken, error) {
	return r.queries.GetPasswordResetToken(ctx, token)
}

func (r *repository) DeletePasswordResetTokenByToken(ctx context.Context, token string) error {
	return r.queries.DeletePasswordResetTokenByToken(ctx, token)
}

func (r *repository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	return r.queries.UpdateCustomerPassword(ctx, dbgen.UpdateCustomerPasswordParams{
		ID:       userID,
		Password: password,
	})
}

func (r *repository) GetLatestEmailConfirmationTokenByUserID(ctx context.Context, userID uuid.UUID) (dbgen.EmailConfirmationToken, error) {
	return r.queries.GetLatestEmailConfirmationTokenByUserID(ctx, userID)
}

func (r *repository) UpsertEmailConfirmationToken(ctx context.Context, params dbgen.UpsertEmailConfirmationTokenParams) error {
	return r.queries.UpsertEmailConfirmationToken(ctx, params)
}

func (r *repository) DeleteEmailConfirmationTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.queries.DeleteEmailConfirmationTokensByUserID(ctx, userID)
}

func (r *repository) GetEmailConfirmationTokenByToken(ctx context.Context, token string) (dbgen.EmailConfirmationToken, error) {
	return r.queries.GetEmailConfirmationTokenByToken(ctx, token)
}

func (r *repository) DeleteEmailConfirmationTokenByToken(ctx context.Context, token string) error {
	return r.queries.DeleteEmailConfirmationTokenByToken(ctx, token)
}

func (r *repository) DeleteEmailConfirmationTokenByPin(ctx context.Context, pin string) error {
	return r.queries.DeleteEmailConfirmationTokenByPin(ctx, pin)
}

func (r *repository) SetUserEmailConfirmed(ctx context.Context, userID uuid.UUID) error {
	return r.queries.SetUserEmailConfirmed(ctx, userID)
}
