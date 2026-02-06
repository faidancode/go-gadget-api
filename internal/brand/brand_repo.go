package brand

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=brand_repo.go -destination=../mock/brand/brand_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	Create(ctx context.Context, arg dbgen.CreateBrandParams) (dbgen.Brand, error)
	ListPublic(ctx context.Context, limit, offset int32) ([]dbgen.ListBrandsPublicRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListBrandsAdminParams) ([]dbgen.ListBrandsAdminRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.Brand, error)
	Update(ctx context.Context, arg dbgen.UpdateBrandParams) (dbgen.Brand, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Brand, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) Create(ctx context.Context, arg dbgen.CreateBrandParams) (dbgen.Brand, error) {
	return r.queries.CreateBrand(ctx, arg)
}

func (r *repository) ListPublic(ctx context.Context, limit, offset int32) ([]dbgen.ListBrandsPublicRow, error) {
	return r.queries.ListBrandsPublic(ctx, dbgen.ListBrandsPublicParams{Limit: limit, Offset: offset})
}

func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListBrandsAdminParams) ([]dbgen.ListBrandsAdminRow, error) {
	return r.queries.ListBrandsAdmin(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.Brand, error) {
	return r.queries.GetBrandByID(ctx, id)
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateBrandParams) (dbgen.Brand, error) {
	return r.queries.UpdateBrand(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteBrand(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (dbgen.Brand, error) {
	return r.queries.RestoreBrand(ctx, id)
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
