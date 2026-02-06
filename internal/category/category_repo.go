package category

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=category_repo.go -destination=../mock/category/category_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	Create(ctx context.Context, arg dbgen.CreateCategoryParams) (dbgen.Category, error)
	ListPublic(ctx context.Context, limit, offset int32) ([]dbgen.ListCategoriesPublicRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListCategoriesAdminParams) ([]dbgen.ListCategoriesAdminRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.Category, error)
	Update(ctx context.Context, arg dbgen.UpdateCategoryParams) (dbgen.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Category, error)
	GetIDsBySlugs(ctx context.Context, slugs []string) ([]uuid.UUID, error)
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

func (r *repository) ListPublic(ctx context.Context, limit, offset int32) ([]dbgen.ListCategoriesPublicRow, error) {
	return r.queries.ListCategoriesPublic(ctx, dbgen.ListCategoriesPublicParams{Limit: limit, Offset: offset})
}

func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListCategoriesAdminParams) ([]dbgen.ListCategoriesAdminRow, error) {
	return r.queries.ListCategoriesAdmin(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.Category, error) {
	return r.queries.GetCategoryByID(ctx, id)
}
func (r *repository) GetIDsBySlugs(ctx context.Context, slugs []string) ([]uuid.UUID, error) {
	if len(slugs) == 0 {
		return []uuid.UUID{}, nil // return empty slice langsung supaya aman
	}
	return r.queries.GetIDsBySlugs(ctx, slugs)
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

func (r *repository) WithTx(tx dbgen.DBTX) Repository {
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &repository{
			queries: r.queries.WithTx(sqlTx),
		}
	}

	// Standard Repo
	return r
}
