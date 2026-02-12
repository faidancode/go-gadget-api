package order_test

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOrderService_Checkout(t *testing.T) {
	// =========================================================
	// SHARED SETUP
	// =========================================================
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})

	ctx := context.Background()

	// =========================================================
	t.Run("success_checkout_single_item", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		orderID := uuid.New()

		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{ProductID: productID.String(), Qty: 2, Price: 5000},
				},
			}, nil).
			Times(1)

		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).Times(1)
		outboxRepo.EXPECT().WithTx(gomock.Any()).Return(outboxRepo).Times(1)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p dbgen.CreateOrderParams) (dbgen.Order, error) {
				assert.NotEmpty(t, p.OrderNumber)
				return dbgen.Order{
					ID:          orderID,
					OrderNumber: p.OrderNumber,
					UserID:      userID,
					Status:      "PENDING",
				}, nil
			}).Times(1)

		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		outboxRepo.EXPECT().
			CreateOutboxEvent(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		res, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})
		require.NoError(t, err)
		assert.NotEmpty(t, res.OrderNumber)

		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	// =========================================================
	t.Run("success_checkout_multiple_items", func(t *testing.T) {
		userID := uuid.New()

		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{ProductID: uuid.NewString(), Qty: 2, Price: 10000},
					{ProductID: uuid.NewString(), Qty: 1, Price: 25000},
					{ProductID: uuid.NewString(), Qty: 3, Price: 5000},
				},
			}, nil).
			Times(1)

		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).Times(1)
		outboxRepo.EXPECT().WithTx(gomock.Any()).Return(outboxRepo).Times(1)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p dbgen.CreateOrderParams) (dbgen.Order, error) {
				assert.Equal(t, "60000.00", p.TotalPrice)
				return dbgen.Order{
					ID:          uuid.New(),
					OrderNumber: p.OrderNumber,
					UserID:      userID,
					Status:      "PENDING",
				}, nil
			}).Times(1)

		orderRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(nil).Times(3)
		outboxRepo.EXPECT().CreateOutboxEvent(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		res, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})
		require.NoError(t, err)
		assert.NotEmpty(t, res.OrderNumber)

		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	// =========================================================
	t.Run("error_empty_cart", func(t *testing.T) {
		userID := uuid.New()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{}, nil).
			Times(1)

		_, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, order.ErrCartEmpty)
	})

	// =========================================================
	t.Run("error_cart_service_failed", func(t *testing.T) {
		userID := uuid.New()
		expectedErr := errors.New("cart down")

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{}, expectedErr).
			Times(1)

		_, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	// =========================================================
	t.Run("error_create_order_failed_should_rollback", func(t *testing.T) {
		userID := uuid.New()

		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{ProductID: uuid.NewString(), Qty: 1, Price: 1000},
				},
			}, nil).Times(1)

		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).Times(1)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{}, order.ErrOrderFailed).
			Times(1)

		_, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, order.ErrOrderFailed)

		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	// =========================================================
	t.Run("error_create_order_item_failed_should_rollback", func(t *testing.T) {
		userID := uuid.New()

		sqlMock.ExpectBegin()
		// Rollback akan terpanggil karena 'committed' masih false saat return error
		sqlMock.ExpectRollback()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{ProductID: uuid.NewString(), Qty: 1, Price: 1000},
				},
			}, nil).Times(1)

		// Setup Repo dengan WithTx
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).Times(1)

		// PENTING: outboxRepo.WithTx JANGAN dipanggil di sini
		// karena CreateOrderItem sudah melempar error dan fungsi langsung return.
		// Jadi eksekusi tidak akan sampai ke bagian Outbox Event.

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{
				ID: uuid.New(), OrderNumber: "ORD-FAIL", UserID: userID, Status: "PENDING",
			}, nil).Times(1)

		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(order.ErrOrderFailed). // Sengaja dibuat error
			Times(1)

		// Execution
		_, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})

		// Assertion
		require.Error(t, err)
		assert.ErrorIs(t, err, order.ErrOrderFailed)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	// =========================================================
	t.Run("error_commit_failed", func(t *testing.T) {
		// -------------------------------------------------
		// Arrange - Test Data
		// -------------------------------------------------
		userID := uuid.New()
		productID := uuid.New()
		orderID := uuid.New()

		// -------------------------------------------------
		// Arrange - Mock Cart Service
		// -------------------------------------------------
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{
						ProductID: productID.String(),
						Qty:       1,
						Price:     1000,
					},
				},
			}, nil)

		// -------------------------------------------------
		// Arrange - Mock Repository (Transactional)
		// -------------------------------------------------
		orderRepo.EXPECT().
			WithTx(gomock.Any()).
			Return(orderRepo)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{
				ID:          orderID,
				OrderNumber: "ORD-001",
				UserID:      userID,
				Status:      "PENDING",
			}, nil)

		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(nil)

		outboxRepo.EXPECT().
			WithTx(gomock.Any()).
			Return(outboxRepo)

		outboxRepo.EXPECT().
			CreateOutboxEvent(gomock.Any(), gomock.Any()).
			Return(nil)

		// -------------------------------------------------
		// Arrange - Mock Database Transaction
		// -------------------------------------------------
		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit().WillReturnError(errors.New("commit failed"))
		// HAPUS: sqlMock.ExpectRollback()
		// Karena commit failed = transaksi sudah terminated di sqlmock

		// -------------------------------------------------
		// Act
		// -------------------------------------------------
		_, err := svc.Checkout(ctx, userID.String(), order.CheckoutRequest{})

		// -------------------------------------------------
		// Assert
		// -------------------------------------------------
		require.Error(t, err)
		assert.ErrorIs(t, err, order.ErrOrderFailed)
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

}

func TestOrderService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})

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

		res, total, err := svc.List(ctx, userID.String(), "ALL", 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
	})

	t.Run("error_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

		_, _, err := svc.List(ctx, userID.String(), "ALL", 1, 10)
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
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})

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
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})
	ctx := context.Background()

	t.Run("success_get_detail", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.GetOrderByIDRow{ID: orderID, OrderNumber: "ORD-123"}, nil)

		res, err := svc.Detail(ctx, orderID.String())
		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
	})

	t.Run("error_order_not_found", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.GetOrderByIDRow{}, sql.ErrNoRows)

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
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})
	ctx := context.Background()

	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New()

		// 1. Mock GetByID (DILUAR/SEBELUM transaksi)
		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(dbgen.GetOrderByIDRow{
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
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.GetOrderByIDRow{
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
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})
	ctx := context.Background()

	t.Run("customer_success_complete", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()
		statusTarget := "COMPLETED"

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		// 1. Mock GetByID: Pastikan UserID sama dan status SHIPPED/DELIVERED
		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.GetOrderByIDRow{
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

		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.GetOrderByIDRow{
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
	outboxRepo := outboxMock.NewMockRepository(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(order.Deps{
		DB:         db,
		Repo:       orderRepo,
		OutboxRepo: outboxRepo,
		CartSvc:    cartSvc,
	})
	ctx := context.Background()

	t.Run("admin_success_processing", func(t *testing.T) {
		orderID := uuid.New()
		statusTarget := "PROCESSING"

		mock.ExpectBegin()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo)

		// 1. Mock GetByID untuk validasi status awal (harus PAID)
		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.GetOrderByIDRow{
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

		orderRepo.EXPECT().GetByID(ctx, orderID).Return(dbgen.GetOrderByIDRow{
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
