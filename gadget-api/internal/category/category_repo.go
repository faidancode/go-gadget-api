package category

import (
	"context"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, arg dbgen.CreateCategoryParams) (dbgen.Category, error)
	List(ctx context.Context, limit, offset int32) ([]dbgen.ListCategoriesRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.Category, error)
	Update(ctx context.Context, arg dbgen.UpdateCategoryParams) (dbgen.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Category, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) Create(ctx context.Context, arg dbgen.CreateCategoryParams) (dbgen.Category, error) {
	return r.queries.CreateCategory(ctx, arg)
}

func (r *repository) List(ctx context.Context, limit, offset int32) ([]dbgen.ListCategoriesRow, error) {
	return r.queries.ListCategories(ctx, dbgen.ListCategoriesParams{Limit: limit, Offset: offset})
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.Category, error) {
	return r.queries.GetCategoryByID(ctx, id)
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateCategoryParams) (dbgen.Category, error) {
	return r.queries.UpdateCategory(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteCategory(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (dbgen.Category, error) {
	return r.queries.RestoreCategory(ctx, id)
}
