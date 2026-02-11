package wishlist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

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

func (s *service) List(ctx context.Context, userID string) (WishlistResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return WishlistResponse{}, fmt.Errorf("invalid user id format: %w", err)
	}

	row, err := s.repo.GetWishlistWithItems(ctx, uid)
	if err != nil {
		log.Printf("Error listing wishlist items: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return WishlistResponse{
				UserID:    userID,
				Items:     []WishlistItemResponse{},
				ItemCount: 0,
			}, nil
		}
		return WishlistResponse{}, fmt.Errorf("failed to get wishlist: %w", err)
	}

	// Empty slice safety
	items := make([]WishlistItemResponse, 0)

	if len(row.Items) > 0 {
		if err := json.Unmarshal(row.Items, &items); err != nil {
			log.Printf("Error unmarshaling wishlist items: %v", err)
			return WishlistResponse{}, fmt.Errorf("failed to unmarshal wishlist items: %w", err)
		}
	}

	// Safety: never return nil slice
	if items == nil {
		items = []WishlistItemResponse{}
	}

	return WishlistResponse{
		ID:        row.ID.String(),
		UserID:    row.UserID.String(),
		Items:     items,
		ItemCount: int(len(items)),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
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
