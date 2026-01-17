package order_test

import (
	"context"
	"database/sql"
	"gadget-api/internal/cart"
	cartMock "gadget-api/internal/cart/mock"
	"gadget-api/internal/dbgen"
	orderMock "gadget-api/internal/mock/order"
	"gadget-api/internal/order"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrderService_Checkout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_checkout", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		orderID := uuid.New()

		// --- SQL Mock Expectations ---
		mock.ExpectBegin()
		mock.ExpectCommit()

		// --- Repo Mock Expectations ---
		// PENTING: Mock WithTx agar tidak mengembalikan nil
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{
						ProductID: productID.String(),
						Qty:       2,
						Price:     5000,
					},
				},
			}, nil)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{
				ID:          orderID,
				OrderNumber: "ORD-123",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
			}, nil)

		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(nil)

		cartSvc.EXPECT().
			Delete(gomock.Any(), userID.String()).
			Return(nil)

		// Execute
		res, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_create_order_failed_should_rollback", func(t *testing.T) {
		userID := uuid.New()

		// --- SQL Mock: Expect Begin and then Rollback because of error ---
		mock.ExpectBegin()
		mock.ExpectRollback()

		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{{ProductID: uuid.New().String(), Qty: 1, Price: 1000}},
			}, nil)

		// Simulate error in DB
		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{}, assert.AnError)

		_, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID: userID.String(),
		})

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_cart_empty", func(t *testing.T) {
		userID := uuid.New()

		// Tidak ada mock.ExpectBegin karena fungsi return sebelum transaksi mulai
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{Items: []cart.CartItemDetailResponse{}}, nil)

		_, err := svc.Checkout(ctx, order.CheckoutRequest{UserID: userID.String()})

		assert.ErrorIs(t, err, order.ErrCartEmpty)
	})
}

func TestOrderService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderID1 := uuid.New()
		orderID2 := uuid.New()

		mockRows := []dbgen.ListOrdersRow{
			{ID: orderID1, OrderNumber: "ORD-001", UserID: userID, Status: "PENDING", TotalPrice: "10000.00", TotalCount: 2},
			{ID: orderID2, OrderNumber: "ORD-002", UserID: userID, Status: "COMPLETED", TotalPrice: "20000.00", TotalCount: 2},
		}

		orderRepo.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(mockRows, nil)

		res, total, err := svc.List(ctx, userID.String(), 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
	})

	t.Run("error_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

		_, _, err := svc.List(ctx, userID.String(), 1, 10)
		assert.Error(t, err)
	})
}

func TestOrderService_ListAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_list_all_orders", func(t *testing.T) {
		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			Return([]dbgen.ListOrdersAdminRow{
				{ID: uuid.New(), OrderNumber: "ORD-001", TotalCount: 1},
			}, nil)

		res, total, err := svc.ListAdmin(ctx, "", "", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
	})
}

func TestOrderService_Detail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_get_detail", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{ID: orderID, OrderNumber: "ORD-123"}, nil)
		orderRepo.EXPECT().GetItems(gomock.Any(), orderID).Return([]dbgen.OrderItem{}, nil)

		res, err := svc.Detail(ctx, orderID.String())
		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
	})

	t.Run("error_order_not_found", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{}, sql.ErrNoRows)

		_, err := svc.Detail(ctx, orderID.String())
		assert.Error(t, err)
	})
}

func TestOrderService_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New()

		// 1. Mock GetByID (DILUAR/SEBELUM transaksi)
		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(dbgen.Order{
				ID: orderID, Status: "PENDING",
			}, nil)

		// 2. Setup Transaction Mock (Setelah GetByID)
		mock.ExpectBegin()

		// 3. Mock WithTx dan UpdateStatus (DIDALAM transaksi)
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()
		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "CANCELLED").
			Return(dbgen.Order{}, nil)

		mock.ExpectCommit()

		// Execute
		err := svc.Cancel(ctx, orderID.String())

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_order_not_pending", func(t *testing.T) {
		orderID := uuid.New()
		// Tidak ada BeginTx karena divalidasi sebelum transaksi (opsional, tergantung logic service Anda)
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{
			ID: orderID, Status: "COMPLETED",
		}, nil)

		err := svc.Cancel(ctx, orderID.String())
		assert.ErrorIs(t, err, order.ErrCannotCancel)
	})
}

func TestOrderService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectCommit()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "COMPLETED").
			Return(dbgen.Order{ID: orderID, Status: "COMPLETED", OrderNumber: "ORD-123"}, nil)

		res, err := svc.UpdateStatus(ctx, orderID.String(), "COMPLETED")

		assert.NoError(t, err)
		assert.Equal(t, "COMPLETED", res.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
