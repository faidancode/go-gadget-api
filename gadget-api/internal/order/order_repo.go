package order

import (
	"context"
	"database/sql"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=order_repo.go -destination=../mock/order/order_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error)
	CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.Order, error)
	GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.OrderItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error)
	List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
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

func (r *repository) CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error) {
	return r.queries.CreateOrder(ctx, arg)
}

func (r *repository) CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error {
	return r.queries.CreateOrderItem(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.Order, error) {
	return r.queries.GetOrderByID(ctx, id)
}

func (r *repository) GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.OrderItem, error) {
	return r.queries.GetOrderItems(ctx, orderID)
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error) {
	return r.queries.UpdateOrderStatus(ctx, dbgen.UpdateOrderStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *repository) List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error) {
	return r.queries.ListOrders(ctx, arg)
}

func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
	return r.queries.ListOrdersAdmin(ctx, arg)
}
