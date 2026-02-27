package cart_test

import (
	"context"
	"go-gadget-api/internal/cart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeCartService struct {
	CreateFn     func(ctx context.Context, userID string) error
	CountFn      func(ctx context.Context, userID string) (int64, error)
	DetailFn     func(ctx context.Context, userID string) (cart.CartDetailResponse, error)
	AddItemFn    func(ctx context.Context, userID string, req cart.AddItemRequest) error
	UpdateQtyFn  func(ctx context.Context, userID, productID string, req cart.UpdateQtyRequest) error
	IncrementFn  func(ctx context.Context, userID, productID string) error
	DecrementFn  func(ctx context.Context, userID, productID string) error
	DeleteItemFn func(ctx context.Context, userID, productID string) error
	DeleteFn     func(ctx context.Context, userID string) error
}

func (f *fakeCartService) Create(ctx context.Context, userID string) error {
	return f.CreateFn(ctx, userID)
}
func (f *fakeCartService) Count(ctx context.Context, userID string) (int64, error) {
	return f.CountFn(ctx, userID)
}
func (f *fakeCartService) Detail(ctx context.Context, userID string) (cart.CartDetailResponse, error) {
	return f.DetailFn(ctx, userID)
}
func (f *fakeCartService) AddItem(ctx context.Context, userID string, req cart.AddItemRequest) error {
	if f.AddItemFn == nil {
		return nil
	}
	return f.AddItemFn(ctx, userID, req)
}
func (f *fakeCartService) UpdateQty(ctx context.Context, userID, productID string, req cart.UpdateQtyRequest) error {
	return f.UpdateQtyFn(ctx, userID, productID, req)
}
func (f *fakeCartService) Increment(ctx context.Context, userID, productID string) error {
	return f.IncrementFn(ctx, userID, productID)
}
func (f *fakeCartService) Decrement(ctx context.Context, userID, productID string) error {
	return f.DecrementFn(ctx, userID, productID)
}
func (f *fakeCartService) DeleteItem(ctx context.Context, userID, productID string) error {
	return f.DeleteItemFn(ctx, userID, productID)
}
func (f *fakeCartService) Delete(ctx context.Context, userID string) error {
	return f.DeleteFn(ctx, userID)
}

func (f *fakeCartService) ClearCart(ctx context.Context, cartID string) error {
	return f.DeleteFn(ctx, cartID)
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestHandler(svc cart.Service) *cart.Handler {
	return cart.NewHandler(svc)
}

// Helper untuk menambahkan cookie auth ke request
func addAuthCookie(req *http.Request) {
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "mocked-jwt-token",
	})
}

// ==================== TEST CASES ====================

func TestCartHandler_Create(t *testing.T) {
	t.Run("success_create_cart", func(t *testing.T) {
		userID := "user-123"
		svc := &fakeCartService{
			CreateFn: func(ctx context.Context, uid string) error {
				// Pastikan assert ini menerima ID yang benar
				assert.Equal(t, userID, uid)
				return nil
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/carts", nil)

		// 1. Set context sesuai dengan yang diharapkan oleh ExtractUserID/Handler
		// Sesuai file extract_user.go Anda:
		c.Set("user_id", userID) // Diperlukan jika Handler pakai c.Get("user_id")
		c.Set("user_id", userID) // Diperlukan jika Handler pakai c.Get("user_id")

		// 2. Jika Handler menggunakan c.Param("userId")
		c.Params = gin.Params{{Key: "userId", Value: userID}}

		ctrl.Create(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestCartHandler_Count(t *testing.T) {
	t.Run("success_get_count", func(t *testing.T) {
		svc := &fakeCartService{
			CountFn: func(ctx context.Context, userID string) (int64, error) {
				return 5, nil
			},
		}

		ctrl := newTestHandler(svc)
		r := setupTestRouter()

		// Simulasi route dengan middleware minimal
		r.GET("/cart/count", func(c *gin.Context) {
			c.Set("user_id", "user-123") // Mock hasil middleware
			ctrl.Count(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/cart/count", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":5`)
	})
}

func TestCartHandler_UpdateQty(t *testing.T) {
	t.Run("success_update_qty", func(t *testing.T) {
		svc := &fakeCartService{
			UpdateQtyFn: func(ctx context.Context, userID, productID string, req cart.UpdateQtyRequest) error {
				assert.Equal(t, int32(2), req.Qty)
				return nil
			},
		}

		ctrl := newTestHandler(svc)
		r := setupTestRouter()
		r.PUT("/cart/items/:productId", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			ctrl.UpdateQty(c)
		})

		body := `{"qty":2}`
		req := httptest.NewRequest(http.MethodPut, "/cart/items/prod-1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthCookie(req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("bad_request_invalid_json", func(t *testing.T) {
		ctrl := newTestHandler(&fakeCartService{})
		r := setupTestRouter()
		r.PUT("/cart/items/:productId", func(c *gin.Context) {
			c.Set("user_id", "user-1")
			ctrl.UpdateQty(c)
		})

		req := httptest.NewRequest(http.MethodPut, "/cart/items/prod-1", strings.NewReader(`{"qty":"invalid"}`))
		req.Header.Set("Content-Type", "application/json")
		addAuthCookie(req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCartHandler_IncrementDecrement(t *testing.T) {
	svc := &fakeCartService{
		IncrementFn: func(ctx context.Context, userID, productID string) error { return nil },
		DecrementFn: func(ctx context.Context, userID, productID string) error { return nil },
	}

	ctrl := newTestHandler(svc)
	r := setupTestRouter()

	// Helper wrapper untuk simulasi auth context
	authWrapper := func(handler gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("user_id", "user-mock")
			handler(c)
		}
	}

	r.POST("/cart/items/:productId/increment", authWrapper(ctrl.Increment))
	r.POST("/cart/items/:productId/decrement", authWrapper(ctrl.Decrement))

	t.Run("success_increment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/cart/items/p1/increment", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success_decrement", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/cart/items/p1/decrement", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCartHandler_Delete(t *testing.T) {
	svc := &fakeCartService{
		DeleteItemFn: func(ctx context.Context, userID, productID string) error { return nil },
		DeleteFn:     func(ctx context.Context, userID string) error { return nil },
	}

	ctrl := newTestHandler(svc)
	r := setupTestRouter()

	r.DELETE("/cart/items/:productId", func(c *gin.Context) {
		c.Set("user_id", "user-mock")
		ctrl.DeleteItem(c)
	})
	r.DELETE("/cart", func(c *gin.Context) {
		c.Set("user_id", "user-mock")
		ctrl.DeleteItem(c)
	})

	t.Run("success_delete_item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/cart/items/p1", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success_delete_cart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/cart", nil)
		addAuthCookie(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
