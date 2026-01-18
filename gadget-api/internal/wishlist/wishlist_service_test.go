package wishlist_test

import (
	"context"
	"database/sql"
	"gadget-api/internal/dbgen"
	wishlistMock "gadget-api/internal/mock/wishlist"
	"gadget-api/internal/wishlist"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWishlistService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := wishlistMock.NewMockRepository(ctrl)
	svc := wishlist.NewService(db, repo)
	ctx := context.Background()

	t.Run("success_add_item", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		// SQL Mock Expectations
		mock.ExpectBegin()
		mock.ExpectCommit()

		// Repo Mock Expectations
		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()

		repo.EXPECT().
			GetOrCreateWishlist(gomock.Any(), userID).
			Return(dbgen.Wishlist{
				ID:        wishlistID,
				UserID:    userID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil)

		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(false, nil)

		repo.EXPECT().
			AddItem(gomock.Any(), wishlistID, productID).
			Return(nil)

		// Execute
		res, err := svc.Create(ctx, userID.String(), productID.String())

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Product added to wishlist successfully", res.Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_item_already_exists", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()

		repo.EXPECT().
			GetOrCreateWishlist(gomock.Any(), userID).
			Return(dbgen.Wishlist{ID: wishlistID, UserID: userID}, nil)

		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(true, nil)

		// Execute
		_, err := svc.Create(ctx, userID.String(), productID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrItemAlreadyExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_invalid_product_id", func(t *testing.T) {
		userID := uuid.New()

		// No transaction expected as validation happens before
		_, err := svc.Create(ctx, userID.String(), "invalid-uuid")

		assert.ErrorIs(t, err, wishlist.ErrInvalidProductID)
	})

	t.Run("error_add_item_failed", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()

		repo.EXPECT().
			GetOrCreateWishlist(gomock.Any(), userID).
			Return(dbgen.Wishlist{ID: wishlistID, UserID: userID}, nil)

		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(false, nil)

		repo.EXPECT().
			AddItem(gomock.Any(), wishlistID, productID).
			Return(assert.AnError)

		// Execute
		_, err := svc.Create(ctx, userID.String(), productID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrWishlistFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWishlistService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	repo := wishlistMock.NewMockRepository(ctrl)
	svc := wishlist.NewService(db, repo)
	ctx := context.Background()

	t.Run("success_list_items", func(t *testing.T) {
		userID := uuid.New()
		wishlistID := uuid.New()
		productID1 := uuid.New()
		productID2 := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{
				ID:        wishlistID,
				UserID:    userID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil)

		repo.EXPECT().
			GetItems(gomock.Any(), wishlistID).
			Return([]dbgen.GetWishlistItemsRow{
				{
					ID:        uuid.New(),
					ProductID: productID1,
					Name:      "Product 1",
					Price:     "100000.00",
					Stock:     10,
					ImageUrl:  sql.NullString{String: "image1.jpg", Valid: true},
					CreatedAt: time.Now(),
				},
				{
					ID:        uuid.New(),
					ProductID: productID2,
					Name:      "Product 2",
					Price:     "200000.00",
					Stock:     5,
					ImageUrl:  sql.NullString{String: "image2.jpg", Valid: true},
					CreatedAt: time.Now(),
				},
			}, nil)

		// Execute
		res, err := svc.List(ctx, userID.String())

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, res.ItemCount)
		assert.Len(t, res.Items, 2)
		assert.Equal(t, "Product 1", res.Items[0].Name)
		assert.Equal(t, float64(100000), res.Items[0].Price)
	})

	t.Run("success_empty_wishlist", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{}, sql.ErrNoRows)

		// Execute
		res, err := svc.List(ctx, userID.String())

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 0, res.ItemCount)
		assert.Empty(t, res.Items)
	})

	t.Run("error_invalid_user_id", func(t *testing.T) {
		_, err := svc.List(ctx, "invalid-uuid")

		assert.ErrorIs(t, err, wishlist.ErrInvalidProductID)
	})

	t.Run("error_get_items_failed", func(t *testing.T) {
		userID := uuid.New()
		wishlistID := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{ID: wishlistID, UserID: userID}, nil)

		repo.EXPECT().
			GetItems(gomock.Any(), wishlistID).
			Return(nil, assert.AnError)

		// Execute
		_, err := svc.List(ctx, userID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrWishlistFailed)
	})
}

func TestWishlistService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := wishlistMock.NewMockRepository(ctrl)
	svc := wishlist.NewService(db, repo)
	ctx := context.Background()

	t.Run("success_delete_item", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		// Mock GetWishlistByUserID (OUTSIDE transaction)
		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{
				ID:     wishlistID,
				UserID: userID,
			}, nil)

		// Mock CheckItemExists (OUTSIDE transaction)
		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(true, nil)

		// Transaction Mock
		mock.ExpectBegin()

		// Mock WithTx and DeleteItem (INSIDE transaction)
		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()
		repo.EXPECT().
			DeleteItem(gomock.Any(), wishlistID, productID).
			Return(nil)

		mock.ExpectCommit()

		// Execute
		err := svc.Delete(ctx, userID.String(), productID.String())

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_wishlist_not_found", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{}, sql.ErrNoRows)

		// Execute
		err := svc.Delete(ctx, userID.String(), productID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrWishlistNotFound)
	})

	t.Run("error_item_not_found", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{ID: wishlistID, UserID: userID}, nil)

		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(false, nil)

		// Execute
		err := svc.Delete(ctx, userID.String(), productID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrItemNotFound)
	})

	t.Run("error_invalid_product_id", func(t *testing.T) {
		userID := uuid.New()

		err := svc.Delete(ctx, userID.String(), "invalid-uuid")

		assert.ErrorIs(t, err, wishlist.ErrInvalidProductID)
	})

	t.Run("error_delete_failed", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		wishlistID := uuid.New()

		repo.EXPECT().
			GetWishlistByUserID(gomock.Any(), userID).
			Return(dbgen.Wishlist{ID: wishlistID, UserID: userID}, nil)

		repo.EXPECT().
			CheckItemExists(gomock.Any(), wishlistID, productID).
			Return(true, nil)

		mock.ExpectBegin()
		mock.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()
		repo.EXPECT().
			DeleteItem(gomock.Any(), wishlistID, productID).
			Return(assert.AnError)

		// Execute
		err := svc.Delete(ctx, userID.String(), productID.String())

		// Assert
		assert.ErrorIs(t, err, wishlist.ErrWishlistFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
