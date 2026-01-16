package cart

import (
	"context"
	"errors"
	"gadget-api/internal/cart/mock"
	"gadget-api/internal/dbgen"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCart_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("success_already_exists", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		// Skenario: GetByUserID berhasil, maka tidak perlu CreateCart
		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		err := svc.Create(ctx, userID.String())
		assert.NoError(t, err)
	})

	t.Run("success_create_new", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		// Skenario: GetByUserID gagal (not found), maka panggil CreateCart
		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, errors.New("not found"))

		repo.EXPECT().
			CreateCart(ctx, userID).
			Return(dbgen.Cart{ID: cartID}, nil)

		err := svc.Create(ctx, userID.String())
		assert.NoError(t, err)
	})

	t.Run("error_invalid_user_id", func(t *testing.T) {
		// Skenario: Format UUID salah
		err := svc.Create(ctx, "invalid-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user id")
	})

	t.Run("error_repository_fail", func(t *testing.T) {
		userID := uuid.New()

		// Skenario: Database error saat mencoba membuat cart
		repo.EXPECT().
			GetByUserID(ctx, userID).
			Return(dbgen.Cart{}, errors.New("not found"))

		repo.EXPECT().
			CreateCart(ctx, userID).
			Return(dbgen.Cart{}, errors.New("db connection lost"))

		err := svc.Create(ctx, userID.String())
		assert.Error(t, err)
		assert.Equal(t, "db connection lost", err.Error())
	})
}

func TestCart_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
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

	t.Run("error_invalid_uuid", func(t *testing.T) {
		count, err := svc.Count(ctx, "invalid-uuid")
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "invalid user id")
	})
}

func TestCart_Detail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		prodID := uuid.New()
		itemID := uuid.New()
		now := time.Now()

		mockRows := []dbgen.GetCartDetailRow{
			{
				ID:         itemID,
				ProductID:  prodID,
				Quantity:   2,
				PriceAtAdd: 5000,
				CreatedAt:  now,
			},
		}

		repo.EXPECT().GetDetail(ctx, userID).Return(mockRows, nil)

		res, err := svc.Detail(ctx, userID.String())
		assert.NoError(t, err)
		assert.Len(t, res.Items, 1)
		assert.Equal(t, itemID.String(), res.Items[0].ID)
		assert.Equal(t, int32(2), res.Items[0].Qty)
	})

	t.Run("error_repo_fail", func(t *testing.T) {
		userID := uuid.New()
		repo.EXPECT().GetDetail(ctx, userID).Return(nil, errors.New("db error"))

		_, err := svc.Detail(ctx, userID.String())
		assert.Error(t, err)
	})
}

func TestCart_UpdateQty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("success_increment", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()
		prodID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)
		repo.EXPECT().UpdateQty(ctx, dbgen.UpdateCartItemQtyParams{
			CartID:    cartID,
			ProductID: prodID,
			Quantity:  1,
		}).Return(dbgen.CartItem{}, nil)

		err := svc.Increment(ctx, userID.String(), prodID.String())
		assert.NoError(t, err)
	})

	t.Run("error_invalid_product_id", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{ID: cartID}, nil)

		err := svc.UpdateQty(ctx, userID.String(), "invalid-prod-id", 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Product Id")
	})
}

func TestCart_GetCart_Logic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("create_cart_if_not_exists", func(t *testing.T) {
		userID := uuid.New()
		newCartID := uuid.New()

		// GetByUserID gagal (asumsi belum punya cart)
		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{}, errors.New("not found"))
		// Maka Service harus memanggil CreateCart
		repo.EXPECT().CreateCart(ctx, userID).Return(dbgen.Cart{ID: newCartID}, nil)

		err := svc.Create(ctx, userID.String())
		assert.NoError(t, err)
	})

	t.Run("error_when_create_cart_fails", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().GetByUserID(ctx, userID).Return(dbgen.Cart{}, errors.New("not found"))
		repo.EXPECT().CreateCart(ctx, userID).Return(dbgen.Cart{}, errors.New("fatal db error"))

		err := svc.Create(ctx, userID.String())
		assert.Error(t, err)
		assert.Equal(t, "fatal db error", err.Error())
	})
}

func TestCart_DeleteOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockRepository(ctrl)
	svc := NewService(repo)
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
