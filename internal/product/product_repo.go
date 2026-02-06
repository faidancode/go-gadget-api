package product

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=product_repo.go -destination=../mock/product/product_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error)
	// Pisahkan List menjadi Public dan Admin sesuai query.sql terbaru
	ListPublic(ctx context.Context, arg dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error)

	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetProductByIDRow, error)
	Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error)

	GetBySlug(ctx context.Context, slug string) (dbgen.GetProductBySlugRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
	return r.queries.CreateProduct(ctx, arg)
}

// Implementasi List untuk Customer (Hanya barang aktif & filter harga/sort)
func (r *repository) ListPublic(ctx context.Context, arg dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error) {
	return r.queries.ListProductsPublic(ctx, arg)
}

// Implementasi List untuk Admin (Semua barang & filter dashboard)
func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error) {
	return r.queries.ListProductsAdmin(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetProductByIDRow, error) {
	return r.queries.GetProductByID(ctx, id)
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (dbgen.GetProductBySlugRow, error) {
	return r.queries.GetProductBySlug(ctx, slug)
}

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error) {
	return r.queries.UpdateProduct(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteProduct(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error) {
	return r.queries.RestoreProduct(ctx, id)
}

func (r *repository) WithTx(tx dbgen.DBTX) Repository {
	// Lakukan type assertion dari interface dbgen.DBTX ke *sql.Tx
	// Karena s.db.BeginTx(ctx, nil) di service menghasilkan *sql.Tx
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &repository{
			queries: r.queries.WithTx(sqlTx),
		}
	}

	// Jika gagal (misal yang dipassing adalah *sql.DB),
	// Anda bisa mengembalikan repository standar atau menangani error-nya
	return r
}
