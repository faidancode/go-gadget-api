package order_test

import (
	"context"
	"database/sql"
	"go-gadget-api/internal/auth"
	"go-gadget-api/internal/cart"
	cartMock "go-gadget-api/internal/mock/cart"
	orderMock "go-gadget-api/internal/mock/order"
	outboxMock "go-gadget-api/internal/mock/outbox"
	"go-gadget-api/internal/order"
	"go-gadget-api/internal/shared/database/dbgen"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
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
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
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
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
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
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
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
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
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

func TestOrderService_UpdateStatusByCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
	ctx := context.Background()

	t.Run("customer_success_complete", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()
		statusTarget := "COMPLETED"

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		// 1. Mock GetByID: Pastikan UserID sama dan status SHIPPED/DELIVERED
		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.Order{
			ID: orderID, UserID: userID, Status: "SHIPPED",
		}, nil)

		orderRepo.EXPECT().UpdateStatus(ctx, orderID, statusTarget).Return(dbgen.Order{
			ID: orderID, Status: statusTarget,
		}, nil)

		mock.ExpectCommit()

		res, err := svc.UpdateStatusByCustomer(ctx, orderID.String(), userID, statusTarget)

		assert.NoError(t, err)
		assert.Equal(t, statusTarget, res.Status)
	})

	t.Run("customer_failed_unauthorized", func(t *testing.T) {
		orderID := uuid.New()
		wrongUserID := uuid.New()
		realOwnerID := uuid.New()

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.Order{
			ID: orderID, UserID: realOwnerID, Status: "SHIPPED",
		}, nil)

		// User yang login (wrongUserID) tidak sama dengan pemilik order (realOwnerID)
		_, err := svc.UpdateStatusByCustomer(ctx, orderID.String(), wrongUserID, "COMPLETED")

		assert.Error(t, err)
		assert.Equal(t, auth.ErrUnauthorized, err) // Sesuai pesan error di service Anda
		mock.ExpectRollback()
	})
}

func TestOrderService_UpdateStatusByAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	outboxSvc := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, outboxSvc, cartSvc)
	ctx := context.Background()

	t.Run("admin_success_processing", func(t *testing.T) {
		orderID := uuid.New()
		statusTarget := "PROCESSING"

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		// 1. Mock GetByID untuk validasi status awal (harus PAID)
		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.Order{
			ID: orderID, Status: "PAID",
		}, nil)

		// 2. Mock UpdateStatus
		orderRepo.EXPECT().UpdateStatus(ctx, orderID, statusTarget).Return(dbgen.Order{
			ID: orderID, Status: statusTarget,
		}, nil)

		mock.ExpectCommit()

		res, err := svc.UpdateStatusByAdmin(ctx, orderID.String(), statusTarget, nil)

		assert.NoError(t, err)
		assert.Equal(t, statusTarget, res.Status)
	})

	t.Run("admin_failed_shipped_no_receipt", func(t *testing.T) {
		orderID := uuid.New()
		statusTarget := "SHIPPED"

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.Order{
			ID: orderID, Status: "PROCESSING",
		}, nil)

		// ReceiptNo nil saat status SHIPPED harus return error
		res, err := svc.UpdateStatusByAdmin(ctx, orderID.String(), statusTarget, nil)

		assert.Error(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, order.ErrReceiptRequired, err)
		mock.ExpectRollback()
	})
}
