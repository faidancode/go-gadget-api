package auth

import (
	"context"
	"gadget-api/internal/dbgen"
)

//go:generate mockgen -source=auth_repo.go -destination=mock/auth_repo_mock.go -package=mock
type Repository interface {
	Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error)
	GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *repository) Create(ctx context.Context, params dbgen.CreateUserParams) (dbgen.CreateUserRow, error) {
	return r.queries.CreateUser(ctx, params)
}
