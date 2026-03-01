package order

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=order_repo.go -destination=../mock/order/order_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error)
	CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetOrderByIDRow, error)
	GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.GetOrderItemsRow, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error)
	UpdateOrderSnapToken(ctx context.Context, arg dbgen.UpdateOrderSnapTokenParams) (dbgen.Order, error)
	List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error)

	// New Payment & Summary Methods
	GetOrderPaymentForUpdateByID(ctx context.Context, id uuid.UUID) (dbgen.GetOrderPaymentForUpdateByIDRow, error)
	GetOrderPaymentForUpdateByOrderNumber(ctx context.Context, orderNumber string) (dbgen.GetOrderPaymentForUpdateByOrderNumberRow, error)
	UpdateOrderPaymentStatus(ctx context.Context, arg dbgen.UpdateOrderPaymentStatusParams) (dbgen.Order, error)
	GetOrderSummaryByOrderNumber(ctx context.Context, orderNumber string) (dbgen.GetOrderSummaryByOrderNumberRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) WithTx(tx dbgen.DBTX) Repository {
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &repository{
			queries: r.queries.WithTx(sqlTx),
		}
	}

	return r
}

func (r *repository) CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error) {
	return r.queries.CreateOrder(ctx, arg)
}

func (r *repository) CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error {
	return r.queries.CreateOrderItem(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.GetOrderByIDRow, error) {
	return r.queries.GetOrderByID(ctx, id)
}

func (r *repository) GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.GetOrderItemsRow, error) {
	return r.queries.GetOrderItems(ctx, orderID)
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error) {
	return r.queries.UpdateOrderStatus(ctx, dbgen.UpdateOrderStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *repository) UpdateOrderSnapToken(ctx context.Context, arg dbgen.UpdateOrderSnapTokenParams) (dbgen.Order, error) {
	return r.queries.UpdateOrderSnapToken(ctx, arg)
}

func (r *repository) List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error) {
	return r.queries.ListOrders(ctx, arg)
}

func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
	return r.queries.ListOrdersAdmin(ctx, arg)
}

func (r *repository) GetOrderPaymentForUpdateByID(ctx context.Context, id uuid.UUID) (dbgen.GetOrderPaymentForUpdateByIDRow, error) {
	return r.queries.GetOrderPaymentForUpdateByID(ctx, id)
}

func (r *repository) GetOrderPaymentForUpdateByOrderNumber(ctx context.Context, orderNumber string) (dbgen.GetOrderPaymentForUpdateByOrderNumberRow, error) {
	return r.queries.GetOrderPaymentForUpdateByOrderNumber(ctx, orderNumber)
}

func (r *repository) UpdateOrderPaymentStatus(ctx context.Context, arg dbgen.UpdateOrderPaymentStatusParams) (dbgen.Order, error) {
	return r.queries.UpdateOrderPaymentStatus(ctx, arg)
}

func (r *repository) GetOrderSummaryByOrderNumber(ctx context.Context, orderNumber string) (dbgen.GetOrderSummaryByOrderNumberRow, error) {
	return r.queries.GetOrderSummaryByOrderNumber(ctx, orderNumber)
}

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (dbgen.GetUserByIDRow, error) {
	return r.queries.GetUserByID(ctx, id)
}
