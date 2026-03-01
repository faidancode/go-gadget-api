package order_test

import (
	"context"
	"errors"
	"fmt"
	"go-gadget-api/internal/midtrans"
	"go-gadget-api/internal/order"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeOrderService struct {
	checkoutFunc                         func(ctx context.Context, userID string, req order.CheckoutRequest) (order.OrderResponse, error)
	listFunc                             func(ctx context.Context, userID string, status string, page, limit int) ([]order.OrderResponse, int64, error)
	detailFunc                           func(ctx context.Context, orderID string) (order.OrderResponse, error)
	cancelFunc                           func(ctx context.Context, orderID string) error
	completeFunc                         func(ctx context.Context, orderID string, userID string, nextStatus string) (order.OrderResponse, error)
	listAdminFunc                        func(ctx context.Context, status string, search string, page, limit int) ([]order.OrderResponse, int64, error)
	updateStatusAdminFunc                func(ctx context.Context, orderID string, status string, receiptNo *string) (order.OrderResponse, error)
	updatePaymentStatusFunc              func(ctx context.Context, orderID string, input order.UpdatePaymentStatusInput) (order.OrderResponse, error)
	updatePaymentStatusByOrderNumberFunc func(ctx context.Context, orderNumber string, input order.UpdatePaymentStatusInput) (order.OrderResponse, error)
	handleMidtransNotificationFunc       func(ctx context.Context, payload order.MidtransNotificationRequest) error
	continuePaymentFunc                  func(ctx context.Context, orderID string, userID string) (*midtrans.CreateTransactionResponse, error)
}

func (f *fakeOrderService) Checkout(ctx context.Context, userID string, req order.CheckoutRequest) (order.OrderResponse, error) {
	if f.checkoutFunc != nil {
		return f.checkoutFunc(ctx, userID, req)
	}
	return order.OrderResponse{}, nil
}
func (f *fakeOrderService) List(ctx context.Context, userID string, status string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, userID, status, page, limit)
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
func (f *fakeOrderService) ListAdmin(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listAdminFunc != nil {
		return f.listAdminFunc(ctx, status, search, page, limit)
	}
	return []order.OrderResponse{}, 0, nil
}
func (f *fakeOrderService) Complete(ctx context.Context, orderID string, userID string, nextStatus string) (order.OrderResponse, error) {
	if f.completeFunc != nil {
		return f.completeFunc(ctx, orderID, userID, nextStatus)
	}
	return order.OrderResponse{}, nil
}
func (f *fakeOrderService) UpdateStatusByAdmin(ctx context.Context, orderID string, status string, receiptNo *string) (order.OrderResponse, error) {
	if f.updateStatusAdminFunc != nil {
		return f.updateStatusAdminFunc(ctx, orderID, status, receiptNo)
	}
	return order.OrderResponse{}, nil
}
func (f *fakeOrderService) UpdatePaymentStatus(ctx context.Context, orderID string, input order.UpdatePaymentStatusInput) (order.OrderResponse, error) {
	if f.updatePaymentStatusFunc != nil {
		return f.updatePaymentStatusFunc(ctx, orderID, input)
	}
	return order.OrderResponse{}, nil
}
func (f *fakeOrderService) UpdatePaymentStatusByOrderNumber(ctx context.Context, orderNumber string, input order.UpdatePaymentStatusInput) (order.OrderResponse, error) {
	if f.updatePaymentStatusByOrderNumberFunc != nil {
		return f.updatePaymentStatusByOrderNumberFunc(ctx, orderNumber, input)
	}
	return order.OrderResponse{}, nil
}
func (f *fakeOrderService) HandleMidtransNotification(ctx context.Context, payload order.MidtransNotificationRequest) error {
	if f.handleMidtransNotificationFunc != nil {
		return f.handleMidtransNotificationFunc(ctx, payload)
	}
	return nil
}

func (f *fakeOrderService) ContinuePayment(ctx context.Context, orderID string, userID string) (*midtrans.CreateTransactionResponse, error) {
	if f.continuePaymentFunc != nil {
		return f.continuePaymentFunc(ctx, orderID, userID)
	}
	return nil, nil
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestHandler(svc order.Service, rdb *redis.Client) *order.Handler {
	return order.NewHandler(svc, rdb, nil)
}

func addAuthCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "valid-mock-token",
	})
}

func TestOrderHandler_Checkout(t *testing.T) {
	t.Run("success_checkout", func(t *testing.T) {
		userID := uuid.New().String()
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, userID string, req order.CheckoutRequest) (order.OrderResponse, error) {
				assert.Equal(t, userID, userID)
				return order.OrderResponse{OrderNumber: "ORD-999", Status: "PENDING"}, nil
			},
		}

		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.POST("/orders", func(c *gin.Context) {
			c.Set("user_id", userID)
			ctrl.Checkout(c)
		})

		body := `{"addressId": "addr-123"}`
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthCookie(req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-999")
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		userID := uuid.New().String()
		ctrl := newTestHandler(&fakeOrderService{}, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{invalid}`))
		c.Set("user_id", userID)

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cart_is_empty", func(t *testing.T) {
		userID := uuid.New().String()
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, userID string, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrCartEmpty
			},
		}
		ctrl := newTestHandler(svc, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"address_id":"a"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", userID)

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service_internal_error", func(t *testing.T) {
		userID := uuid.New().String()
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, userID string, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, errors.New("db error")
			},
		}

		ctrl := newTestHandler(svc, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		addressID := uuid.New().String()

		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(`{"addressId":"`+addressID+`"}`),
		)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", userID)

		ctrl.Checkout(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func TestOrderHandler_List(t *testing.T) {
	t.Run("success_list_orders", func(t *testing.T) {
		userID := uuid.New().String()
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, status string, page, limit int) ([]order.OrderResponse, int64, error) {
				return []order.OrderResponse{{OrderNumber: "ORD-001"}}, 1, nil
			},
		}
		ctrl := newTestHandler(svc, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/orders?page=1&limit=10", nil)
		c.Set("user_id", userID)

		ctrl.List(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-001")
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, status string, page, limit int) ([]order.OrderResponse, int64, error) {
				return nil, 0, errors.New("db error")
			},
		}
		ctrl := newTestHandler(svc, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", "user-1")

		ctrl.List(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== DETAIL & CANCEL TESTS ====================

func TestOrderHandler_Detail(t *testing.T) {
	t.Run("success_get_detail", func(t *testing.T) {
		orderID := uuid.New().String()
		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				return order.OrderResponse{OrderNumber: "ORD-123"}, nil
			},
		}
		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.GET("/orders/:id", ctrl.Detail)

		req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-123")
	})

	t.Run("order_not_found", func(t *testing.T) {
		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrOrderNotFound
			},
		}
		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.GET("/orders/:id", ctrl.Detail)

		req := httptest.NewRequest(http.MethodGet, "/orders/non-existent", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestOrderHandler_Cancel(t *testing.T) {
	t.Run("success_cancel", func(t *testing.T) {
		orderID := uuid.New().String()
		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error { return nil },
		}
		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.PATCH("/orders/:id/cancel", ctrl.Cancel)

		req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error { return order.ErrOrderNotFound },
		}
		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.PATCH("/orders/:id/cancel", ctrl.Cancel)

		req := httptest.NewRequest(http.MethodPatch, "/orders/wrong/cancel", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==================== ADMIN TESTS ====================

func TestOrderHandler_ListAdmin(t *testing.T) {
	t.Run("success_list_admin", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, "SHIPPED", status)
				return []order.OrderResponse{{OrderNumber: "ADM-001"}}, 1, nil
			},
		}
		ctrl := newTestHandler(svc, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?status=SHIPPED&page=1&limit=20", nil)

		ctrl.ListAdmin(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ADM-001")
	})
}

func TestOrderHandler_UpdateStatusByAdmin(t *testing.T) {
	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New().String()
		receipt := "RESI-123"
		svc := &fakeOrderService{
			updateStatusAdminFunc: func(ctx context.Context, id, status string, resi *string) (order.OrderResponse, error) {
				assert.Equal(t, "SHIPPED", status)
				assert.Equal(t, receipt, *resi)
				return order.OrderResponse{Status: "SHIPPED"}, nil
			},
		}
		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.PATCH("/admin/orders/:id/status", ctrl.UpdateStatusByAdmin)

		body := `{"status": "SHIPPED", "receiptNo": "RESI-123"}`
		req := httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthCookie(req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		svc := &fakeOrderService{
			updateStatusAdminFunc: func(ctx context.Context, id, status string, resi *string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrOrderNotFound
			},
		}

		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.PATCH("/admin/orders/:id/status", ctrl.UpdateStatusByAdmin)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/admin/orders/none/status",
			strings.NewReader(`{"status":"SHIPPED","receiptNo":"RESI-1"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		fmt.Println("STATUS:", w.Code)
		fmt.Println("BODY:", w.Body.String())

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

}

func TestOrderHandler_UpdatePaymentStatusByAdmin(t *testing.T) {
	t.Run("success_update_payment_status", func(t *testing.T) {
		orderID := uuid.New().String()
		svc := &fakeOrderService{
			updatePaymentStatusFunc: func(ctx context.Context, id string, input order.UpdatePaymentStatusInput) (order.OrderResponse, error) {
				assert.Equal(t, orderID, id)
				assert.Equal(t, "PAID", input.PaymentStatus)
				assert.Equal(t, "bank_transfer", input.PaymentMethod)
				return order.OrderResponse{ID: id, PaymentStatus: "PAID", Status: "PAID"}, nil
			},
		}

		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.PATCH("/admin/orders/:id/payment-status", ctrl.UpdatePaymentStatusByAdmin)

		body := `{"paymentStatus":"PAID","paymentMethod":"bank_transfer"}`
		req := httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/payment-status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "PAID")
	})
}

func TestOrderHandler_HandleMidtransNotification(t *testing.T) {
	t.Run("success_notification", func(t *testing.T) {
		svc := &fakeOrderService{
			handleMidtransNotificationFunc: func(ctx context.Context, payload order.MidtransNotificationRequest) error {
				assert.Equal(t, "ORD-123", payload.OrderID)
				assert.Equal(t, "settlement", payload.TransactionStatus)
				return nil
			},
		}

		ctrl := newTestHandler(svc, nil)
		r := setupTestRouter()
		r.POST("/midtrans/notification", ctrl.HandleMidtransNotification)

		body := `{
			"order_id":"ORD-123",
			"status_code":"200",
			"gross_amount":"10000.00",
			"signature_key":"sig",
			"transaction_status":"settlement",
			"payment_type":"bank_transfer"
		}`
		req := httptest.NewRequest(http.MethodPost, "/midtrans/notification", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
	})
}
