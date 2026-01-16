package cart

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID string) error
	Count(ctx context.Context, userID string) (int64, error)
	Detail(ctx context.Context, userID string) (CartDetailResponse, error)

	UpdateQty(ctx context.Context, userID, productId string, qty int32) error
	Increment(ctx context.Context, userID, productId string) error
	Decrement(ctx context.Context, userID, productId string) error

	DeleteItem(ctx context.Context, userID, productId string) error
	Delete(ctx context.Context, userID string) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) getCart(ctx context.Context, userID string) (uuid.UUID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id")
	}

	cart, err := s.repo.GetByUserID(ctx, uid)
	if err != nil {
		cart, err = s.repo.CreateCart(ctx, uid)
		if err != nil {
			return uuid.Nil, err
		}
	}
	return cart.ID, nil
}

func (s *service) Create(ctx context.Context, userID string) error {
	_, err := s.getCart(ctx, userID)
	return err
}

func (s *service) Count(ctx context.Context, userID string) (int64, error) {
	cartID, err := s.getCart(ctx, userID)
	if err != nil {
		return 0, err
	}
	return s.repo.Count(ctx, cartID)
}

func (s *service) Detail(ctx context.Context, userID string) (CartDetailResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return CartDetailResponse{}, fmt.Errorf("invalid user id")
	}

	rows, err := s.repo.GetDetail(ctx, uid)
	if err != nil {
		return CartDetailResponse{}, err
	}

	items := make([]CartItemDetailResponse, 0)
	for _, r := range rows {
		items = append(items, CartItemDetailResponse{
			ID:        r.ID.String(),
			ProductID: r.ProductID.String(),
			Qty:       r.Quantity,
			Price:     r.PriceAtAdd,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}

	return CartDetailResponse{Items: items}, nil
}

func (s *service) UpdateQty(ctx context.Context, userID, productId string, qty int32) error {
	cartID, err := s.getCart(ctx, userID)
	if err != nil {
		return err
	}

	prodID, err := uuid.Parse(productId)
	if err != nil {
		return fmt.Errorf("Invalid Product Id")
	}

	_, err = s.repo.UpdateQty(ctx, dbgen.UpdateCartItemQtyParams{
		CartID:    cartID,
		ProductID: prodID,
		Quantity:  qty,
	})

	return err
}

func (s *service) Increment(ctx context.Context, userID, productId string) error {
	return s.UpdateQty(ctx, userID, productId, 1)
}

func (s *service) Decrement(ctx context.Context, userID, productId string) error {
	return s.UpdateQty(ctx, userID, productId, -1)
}

func (s *service) DeleteItem(ctx context.Context, userID, productId string) error {
	cartID, err := s.getCart(ctx, userID)
	if err != nil {
		return err
	}
	bid, _ := uuid.Parse(productId)
	return s.repo.DeleteItem(ctx, cartID, bid)
}

func (s *service) Delete(ctx context.Context, userID string) error {
	cartID, err := s.getCart(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, cartID)
}
