package cart_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go-gadget-api/internal/cart"
	carterrors "go-gadget-api/internal/cart/errors"
	"go-gadget-api/internal/dbgen"
	mock "go-gadget-api/internal/mock/cart"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCart_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	t.Run("success_already_exists", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		err := svc.Create(ctx, userID.String())
		assert.NoError(t, err)
	})

	t.Run("success_create_new", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, sql.ErrNoRows)

		repo.EXPECT().
			CreateCart(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		err := svc.Create(ctx, userID.String())
		assert.NoError(t, err)
	})

	t.Run("error_invalid_user_id", func(t *testing.T) {
		err := svc.Create(ctx, "invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("error_create_cart_fail", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, sql.ErrNoRows)

		repo.EXPECT().
			CreateCart(ctx, userID).
			Return(dbgen.Cart{}, errors.New("db error"))

		err := svc.Create(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCartService_AddItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockDB, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()
		productID := uuid.New()

		mockDB.ExpectBegin()
		mockDB.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{}, sql.ErrNoRows)
		repo.EXPECT().CreateCart(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().AddItem(ctx, gomock.Any()).Return(nil)

		err := svc.AddItem(ctx, userID.String(), cart.AddItemRequest{
			ProductID: productID.String(),
			Qty:       2,
			Price:     10000,
		})

		assert.NoError(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("repo_error_should_rollback", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()

		mockDB.ExpectBegin()
		mockDB.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: uuid.New()}, nil)
		repo.EXPECT().AddItem(ctx, gomock.Any()).Return(assert.AnError)

		err := svc.AddItem(ctx, userID.String(), cart.AddItemRequest{
			ProductID: productID.String(),
			Qty:       1,
			Price:     1000,
		})

		assert.Error(t, err)
		assert.NoError(t, mockDB.ExpectationsWereMet())
	})
}

func TestCartService_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().Count(ctx, cartID).Return(int64(3), nil)

		count, err := svc.Count(ctx, userID.String())
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("cart_not_found", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, sql.ErrNoRows)

		_, err := svc.Count(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCartService_Detail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetDetail(ctx, userID).
			Return([]dbgen.GetCartDetailRow{
				{
					ID:         uuid.New(),
					ProductID:  uuid.New(),
					Quantity:   2,
					PriceAtAdd: 10000,
					CreatedAt:  time.Now(),
				},
			}, nil)

		res, err := svc.Detail(ctx, userID.String())
		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
	})

	t.Run("repo_error", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetDetail(ctx, userID).
			Return(nil, assert.AnError)

		_, err := svc.Detail(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCartService_Increment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockDB, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockDB.ExpectBegin()
		mockDB.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().IncrementQty(ctx, cartID, productID).Return(nil)

		err := svc.Increment(ctx, userID.String(), productID.String())
		assert.NoError(t, err)
	})

	t.Run("item_not_found", func(t *testing.T) {
		mockDB.ExpectBegin()
		mockDB.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().IncrementQty(ctx, cartID, productID).Return(sql.ErrNoRows)

		err := svc.Increment(ctx, userID.String(), productID.String())
		assert.Equal(t, carterrors.ErrCartItemNotFound, err)
	})
}

func TestCartService_Decrement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockDB, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	userID := uuid.New()
	cartID := uuid.New()
	productID := uuid.New()

	t.Run("decrement_to_zero_should_delete", func(t *testing.T) {
		mockDB.ExpectBegin()
		mockDB.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().DecrementQty(ctx, cartID, productID).
			Return(dbgen.CartItem{Quantity: 0}, nil)
		repo.EXPECT().DeleteItem(ctx, cartID, productID).Return(nil)

		err := svc.Decrement(ctx, userID.String(), productID.String())
		assert.NoError(t, err)
	})
}

func TestCartService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockDB, _ := sqlmock.New()
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(db, repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		mockDB.ExpectBegin()
		mockDB.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().Delete(ctx, cartID).Return(nil)

		err := svc.Delete(ctx, userID.String())
		assert.NoError(t, err)
	})
}
