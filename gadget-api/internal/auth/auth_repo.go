package auth

import (
	"context"
	"gadget-api/internal/dbgen"
)

type Repository interface {
	GetByUsername(ctx context.Context, username string) (dbgen.User, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) GetByUsername(ctx context.Context, username string) (dbgen.User, error) {
	return r.queries.GetUserByUsername(ctx, username)
}
