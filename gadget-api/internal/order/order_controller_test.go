package order_test

import (
	"context"
	"errors"
	"gadget-api/internal/order"
	"gadget-api/internal/pkg/apperror"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeOrderService struct {
	checkoutFunc     func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error)
	listFunc         func(ctx context.Context, userID string, page, limit int) ([]order.OrderResponse, int64, error)
	detailFunc       func(ctx context.Context, orderID string) (order.OrderResponse, error)
	cancelFunc       func(ctx context.Context, orderID string) error
	listAdminFunc    func(ctx context.Context, status string, search string, page, limit int) ([]order.OrderResponse, int64, error)
	updateStatusFunc func(ctx context.Context, orderID string, status string) (order.OrderResponse, error)
}

func (f *fakeOrderService) Checkout(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
	if f.checkoutFunc != nil {
		return f.checkoutFunc(ctx, req)
	}
	return order.OrderResponse{}, nil
}

func (f *fakeOrderService) List(ctx context.Context, userID string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, userID, page, limit)
	}
	return []order.OrderResponse{}, 0, nil
}

func (f *fakeOrderService) Detail(ctx context.Context, orderID string) (order.OrderResponse, error) {
	if f.detailFunc != nil {
		return f.detailFunc(ctx, orderID)
	}
	return order.OrderResponse{}, nil
}

func (f *fakeOrderService) Cancel(ctx context.Context, orderID string) error {
	if f.cancelFunc != nil {
		return f.cancelFunc(ctx, orderID)
	}
	return nil
}

func (f *fakeOrderService) ListAdmin(ctx context.Context, status string, search string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listAdminFunc != nil {
		return f.listAdminFunc(ctx, status, search, page, limit)
	}
	return []order.OrderResponse{}, 0, nil
}

func (f *fakeOrderService) UpdateStatus(ctx context.Context, orderID string, status string) (order.OrderResponse, error) {
	if f.updateStatusFunc != nil {
		return f.updateStatusFunc(ctx, orderID, status)
	}
	return order.OrderResponse{}, nil
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestController(svc order.Service) *order.Controller {
	return order.NewController(svc)
}

// ==================== CHECKOUT TESTS ====================

func TestOrderController_Checkout(t *testing.T) {
	t.Run("success_checkout", func(t *testing.T) {
		orderID := uuid.New().String()
		userID := uuid.New().String()

		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				assert.Equal(t, userID, req.UserID)
				assert.Equal(t, "addr-123", req.AddressID)

				return order.OrderResponse{
					ID:          orderID,
					OrderNumber: "ORD-999",
					Status:      "PENDING",
					TotalPrice:  150000.00,
					PlacedAt:    time.Now(),
				}, nil
			},
		}

		ctrl := newTestController(svc)
		r := setupTestRouter()
		r.POST("/orders", ctrl.Checkout)

		body := `{"address_id": "addr-123", "note": "Please deliver in the morning"}`
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Simulate middleware setting user_id
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", userID)

		ctrl.Checkout(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-999")
		assert.Contains(t, w.Body.String(), "PENDING")
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		ctrl := newTestController(&fakeOrderService{})
		r := setupTestRouter()
		r.POST("/orders", func(c *gin.Context) {
			c.Set("user_id", "some-user-id") // Set user_id supaya lolos cek auth
			ctrl.Checkout(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{invalid-json}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing_required_fields", func(t *testing.T) {
		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"note": "test"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cart_is_empty", func(t *testing.T) {
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrCartEmpty
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"address_id": "addr-123"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service_internal_error", func(t *testing.T) {
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, errors.New("database connection failed")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"address_id": "addr-123"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== LIST ORDERS TESTS ====================

func TestOrderController_List(t *testing.T) {
	t.Run("success_list_user_orders", func(t *testing.T) {
		userID := uuid.New().String()

		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, limit)

				orders := []order.OrderResponse{
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-001",
						Status:      "PENDING",
						TotalPrice:  100000.00,
						PlacedAt:    time.Now(),
					},
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-002",
						Status:      "PAID",
						TotalPrice:  200000.00,
						PlacedAt:    time.Now(),
					},
				}
				return orders, 2, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders?page=1&limit=10", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-001")
		assert.Contains(t, w.Body.String(), "ORD-002")
	})

	t.Run("success_with_status_filter", func(t *testing.T) {
		userID := uuid.New().String()

		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				// Note: status filter logic should be in controller layer
				orders := []order.OrderResponse{
					{OrderNumber: "ORD-003", Status: "PAID", TotalPrice: 150000.00},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders?status=PAID", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "PAID")
	})

	t.Run("empty_orders", func(t *testing.T) {
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				return []order.OrderResponse{}, 0, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders", nil)
		c.Set("user_id", uuid.New().String())

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				return nil, 0, errors.New("database error")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders", nil)
		c.Set("user_id", uuid.New().String())

		ctrl.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== DETAIL ORDER TESTS ====================

func TestOrderController_Detail(t *testing.T) {
	t.Run("success_get_order_detail", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				assert.Equal(t, orderID, id)

				return order.OrderResponse{
					ID:          orderID,
					OrderNumber: "ORD-123",
					Status:      "PAID",
					TotalPrice:  250000.00,
					PlacedAt:    time.Now(),
					Items: []order.OrderItemResponse{
						{
							ProductID:    uuid.New().String(),
							NameSnapshot: "Product A",
							UnitPrice:    50000.00,
							Quantity:     2,
							Subtotal:     100000.00,
						},
					},
				}, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Detail(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-123")
		assert.Contains(t, w.Body.String(), "Product A")
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				return order.OrderResponse{}, errors.New("order not found")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Detail(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==================== CANCEL ORDER TESTS ====================

func TestOrderController_Cancel(t *testing.T) {
	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				assert.Equal(t, orderID, id)
				return nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("order_already_cancelled", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				return errors.New("order already cancelled")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				return errors.New("order not found")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==================== ADMIN LIST ORDERS TESTS ====================

func TestOrderController_ListAdmin(t *testing.T) {
	t.Run("success_list_all_orders", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, 1, page)
				assert.Equal(t, 20, limit)

				orders := []order.OrderResponse{
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-ADM-001",
						Status:      "PAID",
						TotalPrice:  300000.00,
						PlacedAt:    time.Now(),
					},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?page=1&limit=20", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-ADM-001")
	})

	t.Run("success_filter_by_user", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, "user-123", search) // search could be userID or email

				orders := []order.OrderResponse{
					{OrderNumber: "ORD-USR-001", Status: "PAID"},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?search=user-123", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success_filter_by_status", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, "SHIPPED", status)

				orders := []order.OrderResponse{
					{OrderNumber: "ORD-SHP-001", Status: "SHIPPED"},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?status=SHIPPED", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ==================== UPDATE STATUS TESTS ====================

func TestOrderController_UpdateStatus(t *testing.T) {
	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				assert.Equal(t, orderID, id)
				assert.Equal(t, "SHIPPED", status)

				return order.OrderResponse{
					ID:          id,
					OrderNumber: "ORD-123",
					Status:      status,
					TotalPrice:  100000.00,
					PlacedAt:    time.Now(),
				}, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": "SHIPPED"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid_status", func(t *testing.T) {
		orderID := uuid.New().String()

		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": ""}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		orderID := uuid.New().String()

		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(`{invalid}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrOrderNotFound
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": "DELIVERED"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid_status_transition", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrInvalidStatusTransition
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status":"PENDING"}`
		c.Request = httptest.NewRequest(
			http.MethodPatch,
			"/admin/orders/"+orderID+"/status",
			strings.NewReader(body),
		)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), apperror.CodeInvalidState)
	})

}
