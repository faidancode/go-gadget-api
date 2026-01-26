package cart

import (
	"context"
	"database/sql"
	"time"

	autherrors "gadget-api/internal/auth/errors"
	carterrors "gadget-api/internal/cart/errors"
	"gadget-api/internal/dbgen"
	producterrors "gadget-api/internal/product/errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

//go:generate mockgen -source=cart_service.go -destination=../mock/cart/cart_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, userID string) error
	Count(ctx context.Context, userID string) (int64, error)
	Detail(ctx context.Context, userID string) (CartDetailResponse, error)

	AddItem(ctx context.Context, userID string, req AddItemRequest) error
	UpdateQty(ctx context.Context, userID, productID string, req UpdateQtyRequest) error

	Increment(ctx context.Context, userID, productID string) error
	Decrement(ctx context.Context, userID, productID string) error

	DeleteItem(ctx context.Context, userID, productID string) error
	Delete(ctx context.Context, userID string) error
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(r Repository) Service {
	return &service{
		repo:     r,
		validate: validator.New(),
	}
}

// ========================
// helpers
// ========================

func (s *service) parseUserID(userID string) (uuid.UUID, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, autherrors.ErrInvalidUserID
	}
	return id, nil
}

func (s *service) parseProductID(productID string) (uuid.UUID, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return uuid.Nil, producterrors.ErrInvalidProductID
	}
	return id, nil
}

func (s *service) getCartOnly(ctx context.Context, uid uuid.UUID) (uuid.UUID, error) {
	cart, err := s.repo.GetByUserID(ctx, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, carterrors.ErrCartNotFound
		}
		return uuid.Nil, err
	}
	return cart.ID, nil
}

func (s *service) getOrCreateCart(ctx context.Context, uid uuid.UUID) (uuid.UUID, error) {
	cart, err := s.repo.GetByUserID(ctx, uid)
	if err == nil {
		return cart.ID, nil
	}

	cart, err = s.repo.CreateCart(ctx, uid)
	if err != nil {
		return uuid.Nil, err
	}
	return cart.ID, nil
}

// ========================
// service methods
// ========================

func (s *service) Create(ctx context.Context, userID string) error {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}
	_, err = s.getOrCreateCart(ctx, uid)
	return err
}

func (s *service) Count(ctx context.Context, userID string) (int64, error) {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return 0, err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return 0, err
	}

	return s.repo.Count(ctx, cartID)
}

func (s *service) Detail(ctx context.Context, userID string) (CartDetailResponse, error) {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return CartDetailResponse{}, err
	}

	rows, err := s.repo.GetDetail(ctx, uid)
	if err != nil {
		return CartDetailResponse{}, err
	}

	items := make([]CartItemDetailResponse, 0, len(rows))
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

func (s *service) AddItem(ctx context.Context, userID string, req AddItemRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return carterrors.MapValidationError(err)
	}

	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	pid, err := s.parseProductID(req.ProductID)
	if err != nil {
		return err
	}

	cartID, err := s.getOrCreateCart(ctx, uid)
	if err != nil {
		return err
	}

	return s.repo.AddItem(ctx, dbgen.AddCartItemParams{
		CartID:     cartID,
		ProductID:  pid,
		Quantity:   req.Qty,
		PriceAtAdd: req.Price,
	})
}

func (s *service) UpdateQty(ctx context.Context, userID, productID string, req UpdateQtyRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return carterrors.MapValidationError(err)
	}

	if req.Qty <= 0 {
		return carterrors.ErrInvalidQty
	}

	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	pid, err := s.parseProductID(productID)
	if err != nil {
		return err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return err
	}

	_, err = s.repo.UpdateQty(ctx, dbgen.UpdateCartItemQtyParams{
		CartID:    cartID,
		ProductID: pid,
		Quantity:  req.Qty,
	})

	if err == sql.ErrNoRows {
		return carterrors.ErrCartItemNotFound
	}

	return err
}

func (s *service) Increment(ctx context.Context, userID, productID string) error {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	pid, err := s.parseProductID(productID)
	if err != nil {
		return err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return err
	}

	// komentar: increment = qty + 1 via UpdateQty
	_, err = s.repo.UpdateQty(ctx, dbgen.UpdateCartItemQtyParams{
		CartID:    cartID,
		ProductID: pid,
		Quantity:  1,
	})

	if err == sql.ErrNoRows {
		return carterrors.ErrCartItemNotFound
	}

	return err
}

func (s *service) Decrement(ctx context.Context, userID, productID string) error {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	pid, err := s.parseProductID(productID)
	if err != nil {
		return err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return err
	}

	// komentar:
	// decrement TIDAK boleh bikin qty <= 0
	item, err := s.repo.UpdateQty(ctx, dbgen.UpdateCartItemQtyParams{
		CartID:    cartID,
		ProductID: pid,
		Quantity:  -1,
	})

	if err == sql.ErrNoRows {
		return carterrors.ErrCartItemNotFound
	}

	if item.Quantity <= 0 {
		return s.repo.DeleteItem(ctx, cartID, pid)
	}

	return nil
}

func (s *service) DeleteItem(ctx context.Context, userID, productID string) error {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	pid, err := s.parseProductID(productID)
	if err != nil {
		return err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return err
	}

	return s.repo.DeleteItem(ctx, cartID, pid)
}

func (s *service) Delete(ctx context.Context, userID string) error {
	uid, err := s.parseUserID(userID)
	if err != nil {
		return err
	}

	cartID, err := s.getCartOnly(ctx, uid)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, cartID)
}
