package product

import (
	"context"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error)
	// Pisahkan List menjadi Public dan Admin sesuai query.sql terbaru
	ListPublic(ctx context.Context, arg dbgen.ListProductsPublicParams) ([]dbgen.ListProductsPublicRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListProductsAdminParams) ([]dbgen.ListProductsAdminRow, error)

	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetProductByIDRow, error)
	Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error)
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

func (r *repository) Update(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error) {
	return r.queries.UpdateProduct(ctx, arg)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteProduct(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (dbgen.Product, error) {
	return r.queries.RestoreProduct(ctx, id)
}
