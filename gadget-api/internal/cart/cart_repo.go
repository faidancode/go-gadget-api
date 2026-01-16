package cart

import (
	"context"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=cart_repo.go -destination=mock/cart_repo_mock.go -package=mock
type Repository interface {
	CreateCart(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error)

	Count(ctx context.Context, cartID uuid.UUID) (int64, error)
	GetDetail(ctx context.Context, userID uuid.UUID) ([]dbgen.GetCartDetailRow, error)

	CreateItem(ctx context.Context, arg dbgen.CreateCartItemParams) (dbgen.CartItem, error)
	UpdateQty(ctx context.Context, arg dbgen.UpdateCartItemQtyParams) (dbgen.CartItem, error)

	DeleteItem(ctx context.Context, cartID, productId uuid.UUID) error
	Delete(ctx context.Context, cartID uuid.UUID) error
}

type repository struct {
	q *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{q: q}
}

func (r *repository) CreateCart(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error) {
	return r.q.CreateCart(ctx, userID)
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Cart, error) {
	return r.q.GetCartByUserID(ctx, userID)
}

func (r *repository) Count(ctx context.Context, cartID uuid.UUID) (int64, error) {
	return r.q.CountCartItems(ctx, cartID)
}

func (r *repository) GetDetail(ctx context.Context, userID uuid.UUID) ([]dbgen.GetCartDetailRow, error) {
	return r.q.GetCartDetail(ctx, userID)
}

func (r *repository) CreateItem(ctx context.Context, arg dbgen.CreateCartItemParams) (dbgen.CartItem, error) {
	return r.q.CreateCartItem(ctx, arg)
}

func (r *repository) UpdateQty(ctx context.Context, arg dbgen.UpdateCartItemQtyParams) (dbgen.CartItem, error) {
	return r.q.UpdateCartItemQty(ctx, arg)
}

func (r *repository) DeleteItem(ctx context.Context, cartID, productId uuid.UUID) error {
	return r.q.DeleteCartItem(ctx, dbgen.DeleteCartItemParams{
		CartID:    cartID,
		ProductID: productId,
	})
}

func (r *repository) Delete(ctx context.Context, cartID uuid.UUID) error {
	return r.q.DeleteCart(ctx, cartID)
}
