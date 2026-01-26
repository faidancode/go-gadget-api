package cart_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"gadget-api/internal/cart"
	carterrors "gadget-api/internal/cart/errors"
	"gadget-api/internal/dbgen"
	mock "gadget-api/internal/mock/cart"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCart_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
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

func TestCart_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
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

	t.Run("error_cart_not_found", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, sql.ErrNoRows)

		_, err := svc.Count(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCart_Detail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		itemID := uuid.New()
		prodID := uuid.New()
		now := time.Now()

		rows := []dbgen.GetCartDetailRow{
			{
				ID:         itemID,
				ProductID:  prodID,
				Quantity:   2,
				PriceAtAdd: 10000,
				CreatedAt:  now,
			},
		}

		repo.EXPECT().GetDetail(ctx, userID).Return(rows, nil)

		res, err := svc.Detail(ctx, userID.String())
		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
	})

	t.Run("error_repo_fail", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetDetail(ctx, userID).
			Return(nil, errors.New("db error"))

		_, err := svc.Detail(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCart_AddItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
	ctx := context.Background()

	t.Run("success_add_item", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()
		prodID := uuid.New()

		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, sql.ErrNoRows)

		repo.EXPECT().
			CreateCart(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		repo.EXPECT().
			AddItem(ctx, gomock.Any()).
			Return(nil)

		err := svc.AddItem(ctx, userID.String(), cart.AddItemRequest{
			ProductID: prodID.String(),
			Qty:       2,
			Price:     10000,
		})

		assert.NoError(t, err)
	})

	t.Run("error_invalid_product_id", func(t *testing.T) {
		err := svc.AddItem(ctx, uuid.New().String(), cart.AddItemRequest{
			ProductID: "invalid",
			Qty:       1,
			Price:     1000,
		})
		assert.Error(t, err)
	})
}

func TestCart_Increment_Decrement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	cartID := uuid.New()
	prodID := uuid.New()

	t.Run("increment_success", func(t *testing.T) {
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().UpdateQty(ctx, gomock.Any()).Return(dbgen.CartItem{}, nil)

		err := svc.Increment(ctx, userID.String(), prodID.String())
		assert.NoError(t, err)
	})

	t.Run("decrement_to_zero_delete_item", func(t *testing.T) {
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)

		repo.EXPECT().
			UpdateQty(ctx, gomock.Any()).
			Return(dbgen.CartItem{Quantity: 0}, nil)

		repo.EXPECT().
			DeleteItem(ctx, cartID, prodID).
			Return(nil)

		err := svc.Decrement(ctx, userID.String(), prodID.String())
		assert.NoError(t, err)
	})

	t.Run("increment_item_not_found", func(t *testing.T) {
		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		repo.EXPECT().
			UpdateQty(ctx, gomock.Any()).
			Return(dbgen.CartItem{}, sql.ErrNoRows)

		err := svc.Increment(ctx, userID.String(), prodID.String())
		assert.Equal(t, carterrors.ErrCartItemNotFound, err)
	})
}

func TestCart_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := cart.NewService(repo)
	ctx := context.Background()

	t.Run("delete_item_success", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()
		prodID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().DeleteItem(ctx, cartID, prodID).Return(nil)

		err := svc.DeleteItem(ctx, userID.String(), prodID.String())
		assert.NoError(t, err)
	})

	t.Run("delete_cart_success", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().Delete(ctx, cartID).Return(nil)

		err := svc.Delete(ctx, userID.String())
		assert.NoError(t, err)
	})
}
