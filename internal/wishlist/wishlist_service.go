package wishlist

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/google/uuid"
)

//go:generate mockgen -source=wishlist_service.go -destination=../mock/wishlist/wishlist_service_mock.go -package=mock
type Service interface {
	Create(ctx context.Context, userID, productID string) (AddItemResponse, error)
	List(ctx context.Context, userID string) (WishlistResponse, error)
	Delete(ctx context.Context, userID, productID string) error
}

type service struct {
	repo Repository
	db   *sql.DB
}

func NewService(db *sql.DB, r Repository) Service {
	return &service{
		db:   db,
		repo: r,
	}
}

// Create adds a product to user's wishlist
func (s *service) Create(ctx context.Context, userID, productID string) (AddItemResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return AddItemResponse{}, ErrInvalidProductID
	}

	pid, err := uuid.Parse(productID)
	if err != nil {
		return AddItemResponse{}, ErrInvalidProductID
	}

	// 1. Mulai Transaksi
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AddItemResponse{}, ErrWishlistFailed
	}
	defer tx.Rollback()

	// 2. Gunakan WithTx
	qtx := s.repo.WithTx(tx)

	// 3. Get or Create Wishlist
	wishlist, err := qtx.GetOrCreateWishlist(ctx, uid)
	if err != nil {
		return AddItemResponse{}, ErrWishlistFailed
	}

	// 4. Check if item already exists
	exists, err := qtx.CheckItemExists(ctx, wishlist.ID, pid)
	if err != nil {
		return AddItemResponse{}, ErrWishlistFailed
	}

	if exists {
		return AddItemResponse{}, ErrItemAlreadyExists
	}

	// 5. Add item to wishlist
	err = qtx.AddItem(ctx, wishlist.ID, pid)
	if err != nil {
		return AddItemResponse{}, ErrWishlistFailed
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		return AddItemResponse{}, ErrWishlistFailed
	}

	return AddItemResponse{
		Message: "Product added to wishlist successfully",
	}, nil
}

// List retrieves all items in user's wishlist
func (s *service) List(ctx context.Context, userID string) (WishlistResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return WishlistResponse{}, ErrInvalidProductID
	}

	// Get wishlist
	wishlist, err := s.repo.GetWishlistByUserID(ctx, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty wishlist if not exists
			return WishlistResponse{
				UserID:    userID,
				Items:     []WishlistItemResponse{},
				ItemCount: 0,
			}, nil
		}
		return WishlistResponse{}, ErrWishlistFailed
	}

	// Get items
	items, err := s.repo.GetItems(ctx, wishlist.ID)
	if err != nil {
		return WishlistResponse{}, ErrWishlistFailed
	}

	// Map to response
	var itemResponses []WishlistItemResponse
	for _, item := range items {
		price, _ := strconv.ParseFloat(item.Price, 64)
		itemResponses = append(itemResponses, WishlistItemResponse{
			ID:        item.ID.String(),
			ProductID: item.ProductID.String(),
			Name:      item.Name,
			Price:     price,
			Stock:     item.Stock,
			ImageURL:  item.ImageUrl.String,
			AddedAt:   item.CreatedAt,
		})
	}

	return WishlistResponse{
		ID:        wishlist.ID.String(),
		UserID:    wishlist.UserID.String(),
		Items:     itemResponses,
		ItemCount: len(itemResponses),
		CreatedAt: wishlist.CreatedAt,
		UpdatedAt: wishlist.UpdatedAt,
	}, nil
}

// Delete removes a product from user's wishlist
func (s *service) Delete(ctx context.Context, userID, productID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidProductID
	}

	pid, err := uuid.Parse(productID)
	if err != nil {
		return ErrInvalidProductID
	}

	// 1. Get wishlist
	wishlist, err := s.repo.GetWishlistByUserID(ctx, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrWishlistNotFound
		}
		return ErrWishlistFailed
	}

	// 2. Check if item exists
	exists, err := s.repo.CheckItemExists(ctx, wishlist.ID, pid)
	if err != nil {
		return ErrWishlistFailed
	}

	if !exists {
		return ErrItemNotFound
	}

	// 3. Mulai Transaksi
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrWishlistFailed
	}
	defer tx.Rollback()

	// 4. Gunakan WithTx
	qtx := s.repo.WithTx(tx)

	// 5. Delete item
	err = qtx.DeleteItem(ctx, wishlist.ID, pid)
	if err != nil {
		return ErrWishlistFailed
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		return ErrWishlistFailed
	}

	return nil
}
