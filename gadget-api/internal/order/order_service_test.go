package order_test

import (
	"context"
	"gadget-api/internal/cart"
	cartMock "gadget-api/internal/cart/mock"
	"gadget-api/internal/dbgen"
	orderMock "gadget-api/internal/mock/order"
	"gadget-api/internal/order"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrderService_Checkout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_checkout", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		orderID := uuid.New()

		// 1. Mock cart detail - mengembalikan cart dengan item
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
			}, nil).
			Times(1)

		// 2. Mock create order
		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.CreateOrderParams) (dbgen.Order, error) {
				// Validasi params jika diperlukan
				assert.Equal(t, userID, params.UserID)
				assert.Equal(t, "PENDING", params.Status)

				return dbgen.Order{
					ID:          orderID,
					OrderNumber: "ORD-123",
					UserID:      userID,
					Status:      "PENDING",
					TotalPrice:  "10000.00",
				}, nil
			}).
			Times(1)

		// 3. Mock create order item - dipanggil 1x karena ada 1 item di cart
		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.CreateOrderItemParams) error {
				// Validasi params
				assert.Equal(t, orderID, params.OrderID)
				assert.Equal(t, productID, params.ProductID)
				assert.Equal(t, int32(2), params.Quantity)
				return nil
			}).
			Times(1)

		// 4. Mock cart delete - dipanggil setelah checkout berhasil
		cartSvc.EXPECT().
			Delete(gomock.Any(), userID.String()).
			Return(nil).
			Times(1)

		// Execute
		res, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
			Note:      "Please deliver before 5 PM",
		})

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "ORD-123", res.OrderNumber)
		assert.Equal(t, "PENDING", res.Status)
		assert.Equal(t, float64(10000), res.TotalPrice)
	})

	t.Run("error_cart_empty", func(t *testing.T) {
		userID := uuid.New()

		// Mock empty cart
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{},
			}, nil).
			Times(1)

		// Execute
		_, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, order.ErrCartEmpty)
	})

	t.Run("error_cart_detail_failed", func(t *testing.T) {
		userID := uuid.New()

		// Mock cart detail error
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{}, assert.AnError).
			Times(1)

		// Execute
		_, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, order.ErrCartEmpty)
	})

	t.Run("error_create_order_failed", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()

		// Mock cart detail
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{
						ProductID: productID.String(),
						Qty:       1,
						Price:     5000,
					},
				},
			}, nil).
			Times(1)

		// Mock create order error
		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{}, assert.AnError).
			Times(1)

		// Execute
		_, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		// Assert
		assert.Error(t, err)
	})

	t.Run("success_checkout_multiple_items", func(t *testing.T) {
		userID := uuid.New()
		productID1 := uuid.New()
		productID2 := uuid.New()
		orderID := uuid.New()

		// Mock cart detail dengan multiple items
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{
						ProductID: productID1.String(),
						Qty:       2,
						Price:     5000,
					},
					{
						ProductID: productID2.String(),
						Qty:       1,
						Price:     15000,
					},
				},
			}, nil).
			Times(1)

		// Mock create order
		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{
				ID:          orderID,
				OrderNumber: "ORD-456",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "25000.00", // (2 * 5000) + (1 * 15000)
			}, nil).
			Times(1)

		// Mock create order item - dipanggil 2x karena ada 2 item
		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(2)

		// Mock cart delete
		cartSvc.EXPECT().
			Delete(gomock.Any(), userID.String()).
			Return(nil).
			Times(1)

		// Execute
		res, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "ORD-456", res.OrderNumber)
		assert.Equal(t, float64(25000), res.TotalPrice)
	})
}

func TestOrderService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderID1 := uuid.New()
		orderID2 := uuid.New()

		mockRows := []dbgen.ListOrdersRow{
			{
				ID:          orderID1,
				OrderNumber: "ORD-001",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
				PlacedAt:    time.Now(),
				CreatedAt:   time.Now(),
				TotalCount:  2,
			},
			{
				ID:          orderID2,
				OrderNumber: "ORD-002",
				UserID:      userID,
				Status:      "COMPLETED",
				TotalPrice:  "20000.00",
				PlacedAt:    time.Now(),
				CreatedAt:   time.Now(),
				TotalCount:  2,
			},
		}

		orderRepo.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error) {
				assert.Equal(t, userID, params.UserID)
				assert.Equal(t, int32(10), params.Limit)
				assert.Equal(t, int32(0), params.Offset)
				return mockRows, nil
			}).
			Times(1)

		res, total, err := svc.List(ctx, userID.String(), 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
		assert.Equal(t, "ORD-001", res[0].OrderNumber)
		assert.Equal(t, "PENDING", res[0].Status)
		assert.Equal(t, "ORD-002", res[1].OrderNumber)
		assert.Equal(t, "COMPLETED", res[1].Status)
	})

	t.Run("success_list_orders_page_2", func(t *testing.T) {
		userID := uuid.New()

		orderRepo.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error) {
				assert.Equal(t, int32(10), params.Offset) // page 2 = offset 10
				return []dbgen.ListOrdersRow{}, nil
			}).
			Times(1)

		res, total, err := svc.List(ctx, userID.String(), 2, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, res, 0)
	})

	t.Run("error_list_orders", func(t *testing.T) {
		userID := uuid.New()

		orderRepo.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError).
			Times(1)

		_, _, err := svc.List(ctx, userID.String(), 1, 10)

		assert.Error(t, err)
	})
}

func TestOrderService_ListAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_list_all_orders", func(t *testing.T) {
		orderID1 := uuid.New()
		orderID2 := uuid.New()
		userID1 := uuid.New()
		userID2 := uuid.New()

		mockRows := []dbgen.ListOrdersAdminRow{
			{
				ID:          orderID1,
				OrderNumber: "ORD-001",
				UserID:      userID1,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
				PlacedAt:    time.Now(),
				TotalCount:  2,
			},
			{
				ID:          orderID2,
				OrderNumber: "ORD-002",
				UserID:      userID2,
				Status:      "COMPLETED",
				TotalPrice:  "20000.00",
				PlacedAt:    time.Now(),
				TotalCount:  2,
			},
		}

		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
				assert.Equal(t, int32(10), params.Limit)
				assert.Equal(t, int32(0), params.Offset)
				return mockRows, nil
			}).
			Times(1)

		res, total, err := svc.ListAdmin(ctx, "", "", 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
		assert.Equal(t, "ORD-001", res[0].OrderNumber)
		assert.Equal(t, "ORD-002", res[1].OrderNumber)
	})

	t.Run("success_list_orders_by_status", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockRows := []dbgen.ListOrdersAdminRow{
			{
				ID:          orderID,
				OrderNumber: "ORD-001",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
				PlacedAt:    time.Now(),
				TotalCount:  1,
			},
		}

		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
				assert.True(t, params.Status.Valid)
				assert.Equal(t, "PENDING", params.Status.String)
				return mockRows, nil
			}).
			Times(1)

		res, total, err := svc.ListAdmin(ctx, "PENDING", "", 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
		assert.Equal(t, "PENDING", res[0].Status)
	})

	t.Run("success_list_orders_by_search", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockRows := []dbgen.ListOrdersAdminRow{
			{
				ID:          orderID,
				OrderNumber: "ORD-001",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
				PlacedAt:    time.Now(),
				TotalCount:  1,
			},
		}

		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, params dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
				assert.True(t, params.Search.Valid)
				assert.Equal(t, "ORD-001", params.Search.String)
				return mockRows, nil
			}).
			Times(1)

		res, total, err := svc.ListAdmin(ctx, "", "ORD-001", 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
		assert.Equal(t, "ORD-001", res[0].OrderNumber)
	})

	t.Run("error_list_admin", func(t *testing.T) {
		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError).
			Times(1)

		_, _, err := svc.ListAdmin(ctx, "", "", 1, 10)

		assert.Error(t, err)
	})
}

func TestOrderService_Detail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_get_detail", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()
		productID1 := uuid.New()
		productID2 := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "PENDING",
			TotalPrice:  "25000.00",
			PlacedAt:    time.Now(),
			CreatedAt:   time.Now(),
		}

		mockItems := []dbgen.OrderItem{
			{
				ID:           uuid.New(),
				OrderID:      orderID,
				ProductID:    productID1,
				NameSnapshot: "Product A",
				UnitPrice:    "5000.00",
				Quantity:     2,
				TotalPrice:   "10000.00",
			},
			{
				ID:           uuid.New(),
				OrderID:      orderID,
				ProductID:    productID2,
				NameSnapshot: "Product B",
				UnitPrice:    "15000.00",
				Quantity:     1,
				TotalPrice:   "15000.00",
			},
		}

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(mockOrder, nil).
			Times(1)

		orderRepo.EXPECT().
			GetItems(gomock.Any(), orderID).
			Return(mockItems, nil).
			Times(1)

		res, err := svc.Detail(ctx, orderID.String())

		assert.NoError(t, err)
		assert.Equal(t, orderID.String(), res.ID)
		assert.Equal(t, "ORD-123", res.OrderNumber)
		assert.Equal(t, "PENDING", res.Status)
		assert.Equal(t, float64(25000), res.TotalPrice)
		assert.Len(t, res.Items, 2)
		assert.Equal(t, "Product A", res.Items[0].NameSnapshot)
		assert.Equal(t, int32(2), res.Items[0].Quantity)
		assert.Equal(t, "Product B", res.Items[1].NameSnapshot)
	})

	t.Run("error_invalid_order_id", func(t *testing.T) {
		_, err := svc.Detail(ctx, "invalid-uuid")

		assert.Error(t, err)
	})

	t.Run("error_order_not_found", func(t *testing.T) {
		orderID := uuid.New()

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(dbgen.Order{}, assert.AnError).
			Times(1)

		_, err := svc.Detail(ctx, orderID.String())

		assert.Error(t, err)
	})

	t.Run("error_get_items_failed", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "PENDING",
			TotalPrice:  "25000.00",
			PlacedAt:    time.Now(),
		}

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(mockOrder, nil).
			Times(1)

		orderRepo.EXPECT().
			GetItems(gomock.Any(), orderID).
			Return(nil, assert.AnError).
			Times(1)

		_, err := svc.Detail(ctx, orderID.String())

		assert.Error(t, err)
	})
}

func TestOrderService_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "PENDING",
			TotalPrice:  "10000.00",
			PlacedAt:    time.Now(),
		}

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(mockOrder, nil).
			Times(1)

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "CANCELLED").
			Return(dbgen.Order{
				ID:          orderID,
				OrderNumber: "ORD-123",
				UserID:      userID,
				Status:      "CANCELLED",
				TotalPrice:  "10000.00",
			}, nil).
			Times(1)

		err := svc.Cancel(ctx, orderID.String())

		assert.NoError(t, err)
	})

	t.Run("error_order_not_pending", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "COMPLETED",
			TotalPrice:  "10000.00",
		}

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(mockOrder, nil).
			Times(1)

		err := svc.Cancel(ctx, orderID.String())

		assert.Error(t, err)
		assert.ErrorIs(t, err, order.ErrCannotCancel)
	})

	t.Run("error_order_not_found", func(t *testing.T) {
		orderID := uuid.New()

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(dbgen.Order{}, assert.AnError).
			Times(1)

		err := svc.Cancel(ctx, orderID.String())

		assert.Error(t, err)
	})

	t.Run("error_update_status_failed", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "PENDING",
			TotalPrice:  "10000.00",
		}

		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(mockOrder, nil).
			Times(1)

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "CANCELLED").
			Return(dbgen.Order{}, assert.AnError).
			Times(1)

		err := svc.Cancel(ctx, orderID.String())

		assert.Error(t, err)
	})
}

func TestOrderService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	svc := order.NewService(orderRepo, cartSvc)

	ctx := context.Background()

	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-123",
			UserID:      userID,
			Status:      "COMPLETED",
			TotalPrice:  "10000.00",
			PlacedAt:    time.Now(),
			CreatedAt:   time.Now(),
		}

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "COMPLETED").
			Return(mockOrder, nil).
			Times(1)

		res, err := svc.UpdateStatus(ctx, orderID.String(), "COMPLETED")

		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
		assert.Equal(t, "COMPLETED", res.Status)
	})

	t.Run("success_update_to_shipped", func(t *testing.T) {
		orderID := uuid.New()
		userID := uuid.New()

		mockOrder := dbgen.Order{
			ID:          orderID,
			OrderNumber: "ORD-456",
			UserID:      userID,
			Status:      "SHIPPED",
			TotalPrice:  "20000.00",
			PlacedAt:    time.Now(),
		}

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "SHIPPED").
			Return(mockOrder, nil).
			Times(1)

		res, err := svc.UpdateStatus(ctx, orderID.String(), "SHIPPED")

		assert.NoError(t, err)
		assert.Equal(t, "SHIPPED", res.Status)
	})

	t.Run("error_invalid_order_id", func(t *testing.T) {
		_, err := svc.UpdateStatus(ctx, "invalid-uuid", "COMPLETED")

		assert.Error(t, err)
		assert.ErrorIs(t, err, order.ErrInvalidOrderID)

	})

	t.Run("error_update_failed", func(t *testing.T) {
		orderID := uuid.New()

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "COMPLETED").
			Return(dbgen.Order{}, assert.AnError).
			Times(1)

		_, err := svc.UpdateStatus(ctx, orderID.String(), "COMPLETED")

		assert.Error(t, err)
	})
}
