package wishlist

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=wishlist_repo.go -destination=../mock/wishlist/wishlist_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	GetOrCreateWishlist(ctx context.Context, userID uuid.UUID) (dbgen.Wishlist, error)
	GetWishlistByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Wishlist, error)
	GetWishlistWithItems(ctx context.Context, userID uuid.UUID) (dbgen.GetWishlistWithItemsRow, error)
	AddItem(ctx context.Context, wishlistID, productID uuid.UUID) error
	GetItems(ctx context.Context, wishlistID uuid.UUID) ([]dbgen.GetWishlistItemsRow, error)
	DeleteItem(ctx context.Context, wishlistID, productID uuid.UUID) error
	CheckItemExists(ctx context.Context, wishlistID, productID uuid.UUID) (bool, error)
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

func (r *repository) GetOrCreateWishlist(ctx context.Context, userID uuid.UUID) (dbgen.Wishlist, error) {
	return r.queries.GetOrCreateWishlist(ctx, userID)
}

func (r *repository) GetWishlistByUserID(ctx context.Context, userID uuid.UUID) (dbgen.Wishlist, error) {
	return r.queries.GetWishlistByUserID(ctx, userID)
}

func (r *repository) GetWishlistWithItems(ctx context.Context, userID uuid.UUID) (dbgen.GetWishlistWithItemsRow, error) {
	return r.queries.GetWishlistWithItems(ctx, userID)
}

func (r *repository) AddItem(ctx context.Context, wishlistID, productID uuid.UUID) error {
	return r.queries.AddWishlistItem(ctx, dbgen.AddWishlistItemParams{
		WishlistID: wishlistID,
		ProductID:  productID,
	})
}

func (r *repository) GetItems(ctx context.Context, wishlistID uuid.UUID) ([]dbgen.GetWishlistItemsRow, error) {
	return r.queries.GetWishlistItems(ctx, wishlistID)
}

func (r *repository) DeleteItem(ctx context.Context, wishlistID, productID uuid.UUID) error {
	return r.queries.DeleteWishlistItem(ctx, dbgen.DeleteWishlistItemParams{
		WishlistID: wishlistID,
		ProductID:  productID,
	})
}

func (r *repository) CheckItemExists(ctx context.Context, wishlistID, productID uuid.UUID) (bool, error) {
	return r.queries.CheckWishlistItemExists(ctx, dbgen.CheckWishlistItemExistsParams{
		WishlistID: wishlistID,
		ProductID:  productID,
	})
}
