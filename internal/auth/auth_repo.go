package auth

import (
	"context"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=auth_repo.go -destination=../mock/auth/auth_repo_mock.go -package=mock
type Repository interface {
	Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error)
	GetByEmail(ctx context.Context, email string) (dbgen.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.User, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) GetByEmail(ctx context.Context, email string) (dbgen.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *repository) Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error) {
	return r.queries.CreateUser(ctx, params)
}
