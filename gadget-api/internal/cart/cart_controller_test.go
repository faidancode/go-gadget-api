package cart

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeCartService struct {
	CreateFn func(ctx context.Context, userID string) error
	CountFn  func(ctx context.Context, userID string) (int64, error)
	DetailFn func(ctx context.Context, userID string) (CartDetailResponse, error)

	UpdateQtyFn func(ctx context.Context, userID, bookID string, qty int32) error
	IncrementFn func(ctx context.Context, userID, bookID string) error
	DecrementFn func(ctx context.Context, userID, bookID string) error

	DeleteItemFn func(ctx context.Context, userID, bookID string) error
	DeleteFn     func(ctx context.Context, userID string) error
}

func (f *fakeCartService) Create(ctx context.Context, userID string) error {
	return f.CreateFn(ctx, userID)
}

func (f *fakeCartService) Count(ctx context.Context, userID string) (int64, error) {
	return f.CountFn(ctx, userID)
}

func (f *fakeCartService) Detail(ctx context.Context, userID string) (CartDetailResponse, error) {
	return f.DetailFn(ctx, userID)
}

func (f *fakeCartService) UpdateQty(ctx context.Context, userID, bookID string, qty int32) error {
	return f.UpdateQtyFn(ctx, userID, bookID, qty)
}

func (f *fakeCartService) Increment(ctx context.Context, userID, bookID string) error {
	return f.IncrementFn(ctx, userID, bookID)
}

func (f *fakeCartService) Decrement(ctx context.Context, userID, bookID string) error {
	return f.DecrementFn(ctx, userID, bookID)
}

func (f *fakeCartService) DeleteItem(ctx context.Context, userID, bookID string) error {
	return f.DeleteItemFn(ctx, userID, bookID)
}

func (f *fakeCartService) Delete(ctx context.Context, userID string) error {
	return f.DeleteFn(ctx, userID)
}

func TestCartController_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success_create", func(t *testing.T) {
		svc := &fakeCartService{
			CreateFn: func(ctx context.Context, userID string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.POST("/cart/:userId", ctrl.Create)

		req := httptest.NewRequest(http.MethodPost, "/cart/user-123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestCartController_Count(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		svc := &fakeCartService{
			CountFn: func(ctx context.Context, userID string) (int64, error) {
				return 5, nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.GET("/cart/:userId/count", ctrl.Count)

		req := httptest.NewRequest(http.MethodGet, "/cart/user-123/count", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":5`)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeCartService{
			CountFn: func(ctx context.Context, userID string) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.GET("/cart/:userId/count", ctrl.Count)

		req := httptest.NewRequest(http.MethodGet, "/cart/user-123/count", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCartController_Detail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		svc := &fakeCartService{
			DetailFn: func(ctx context.Context, userID string) (CartDetailResponse, error) {
				return CartDetailResponse{
					Items: []CartItemDetailResponse{
						{ID: "1", ProductID: "prod-1", Qty: 2},
					},
				}, nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.GET("/cart/:userId", ctrl.Detail)

		req := httptest.NewRequest(http.MethodGet, "/cart/user-123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "prod-1")
	})
}

func TestCartController_UpdateQty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success_update", func(t *testing.T) {
		svc := &fakeCartService{
			UpdateQtyFn: func(ctx context.Context, userID, productID string, qty int32) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.PUT("/cart/:userId/items/:productId", ctrl.UpdateQty)

		// Contoh body JSON jika controller memerlukan bind JSON
		body := `{"qty": 10}`
		req := httptest.NewRequest(http.MethodPut, "/cart/user-123/items/prod-99", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("bad_request_payload", func(t *testing.T) {
		ctrl := NewController(&fakeCartService{})
		r := gin.New()
		r.PUT("/cart/:userId/items/:productId", ctrl.UpdateQty)

		body := `{"qty": "not-a-number"}` // Salah tipe data
		req := httptest.NewRequest(http.MethodPut, "/cart/user-123/items/prod-99", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCartController_IncrementDecrement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("increment_success", func(t *testing.T) {
		svc := &fakeCartService{
			IncrementFn: func(ctx context.Context, userID, productID string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.POST("/cart/:userId/items/:productId/increment", ctrl.Increment)

		req := httptest.NewRequest(http.MethodPost, "/cart/user-123/items/prod-1/increment", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("decrement_success", func(t *testing.T) {
		svc := &fakeCartService{
			DecrementFn: func(ctx context.Context, userID, productID string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.POST("/cart/:userId/items/:productId/decrement", ctrl.Decrement)

		req := httptest.NewRequest(http.MethodPost, "/cart/user-123/items/prod-1/decrement", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCartController_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("delete_item_success", func(t *testing.T) {
		svc := &fakeCartService{
			DeleteItemFn: func(ctx context.Context, userID, productID string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.DELETE("/cart/:userId/items/:productId", ctrl.DeleteItem)

		req := httptest.NewRequest(http.MethodDelete, "/cart/user-123/items/prod-1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("clear_cart_success", func(t *testing.T) {
		svc := &fakeCartService{
			DeleteFn: func(ctx context.Context, userID string) error {
				return nil
			},
		}
		ctrl := NewController(svc)
		r := gin.New()
		r.DELETE("/cart/:userId", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/cart/user-123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
