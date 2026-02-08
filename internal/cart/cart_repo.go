package cart

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=cart_repo.go -destination=../mock/cart/cart_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository

	CreateCart(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error)

	Count(ctx context.Context, cartID uuid.UUID) (int64, error)
	GetDetail(ctx context.Context, userID uuid.UUID) ([]dbgen.GetCartDetailRow, error)

	// ⬇️ TAMBAHAN
	GetItemByCartAndProduct(ctx context.Context, cartID, productID uuid.UUID) (dbgen.CartItem, error)

	AddItem(ctx context.Context, arg dbgen.AddCartItemParams) error
	UpdateQty(ctx context.Context, arg dbgen.UpdateCartItemQtyParams) (dbgen.CartItem, error)
	IncrementQty(ctx context.Context, cartID, productID uuid.UUID) (dbgen.CartItem, error)
	DecrementQty(ctx context.Context, cartID, productID uuid.UUID) (dbgen.CartItem, error)

	DeleteItem(ctx context.Context, cartID, productID uuid.UUID) error
	Delete(ctx context.Context, cartID uuid.UUID) error
	DeleteAllItems(ctx context.Context, cartID uuid.UUID) error
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

func (r *repository) CreateCart(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error) {
	return r.queries.CreateCart(ctx, userID)
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error) {
	return r.queries.GetCartByUserID(ctx, userID)
}

func (r *repository) Count(ctx context.Context, cartID uuid.UUID) (int64, error) {
	return r.queries.CountCartItems(ctx, cartID)
}

func (r *repository) GetDetail(ctx context.Context, userID uuid.UUID) ([]dbgen.GetCartDetailRow, error) {
	return r.queries.GetCartDetail(ctx, userID)
}

func (r *repository) GetItemByCartAndProduct(
	ctx context.Context,
	cartID, productID uuid.UUID,
) (dbgen.CartItem, error) {
	return r.queries.GetCartItemByCartAndProduct(ctx, dbgen.GetCartItemByCartAndProductParams{
		CartID:    cartID,
		ProductID: productID,
	})
}

func (r *repository) AddItem(ctx context.Context, arg dbgen.AddCartItemParams) error {
	return r.queries.AddCartItem(ctx, arg)
}

func (r *repository) UpdateQty(ctx context.Context, arg dbgen.UpdateCartItemQtyParams) (dbgen.CartItem, error) {
	return r.queries.UpdateCartItemQty(ctx, arg)
}

func (r *repository) IncrementQty(
	ctx context.Context,
	cartID, productID uuid.UUID,
) (dbgen.CartItem, error) {
	return r.queries.IncrementCartItemQty(ctx, dbgen.IncrementCartItemQtyParams{
		CartID:    cartID,
		ProductID: productID,
	})
}

func (r *repository) DecrementQty(
	ctx context.Context,
	cartID, productID uuid.UUID,
) (dbgen.CartItem, error) {
	return r.queries.DecrementCartItemQty(ctx, dbgen.DecrementCartItemQtyParams{
		CartID:    cartID,
		ProductID: productID,
	})
}

func (r *repository) DeleteItem(ctx context.Context, cartID, productID uuid.UUID) error {
	return r.queries.DeleteCartItem(ctx, dbgen.DeleteCartItemParams{
		CartID:    cartID,
		ProductID: productID,
	})
}

func (r *repository) Delete(ctx context.Context, cartID uuid.UUID) error {
	return r.queries.DeleteCart(ctx, cartID)
}

func (r *repository) DeleteAllItems(ctx context.Context, cartID uuid.UUID) error {
	return r.queries.DeleteAllCartItems(ctx, cartID)
}
