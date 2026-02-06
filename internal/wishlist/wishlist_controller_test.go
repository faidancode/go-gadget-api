package wishlist_test

import (
	"context"
	"errors"
	"go-gadget-api/internal/wishlist"
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

type fakeWishlistService struct {
	createFunc func(ctx context.Context, userID, productID string) (wishlist.AddItemResponse, error)
	listFunc   func(ctx context.Context, userID string) (wishlist.WishlistResponse, error)
	deleteFunc func(ctx context.Context, userID, productID string) error
}

func (f *fakeWishlistService) Create(ctx context.Context, userID, productID string) (wishlist.AddItemResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(ctx, userID, productID)
	}
	return wishlist.AddItemResponse{}, nil
}

func (f *fakeWishlistService) List(ctx context.Context, userID string) (wishlist.WishlistResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, userID)
	}
	return wishlist.WishlistResponse{}, nil
}

func (f *fakeWishlistService) Delete(ctx context.Context, userID, productID string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, userID, productID)
	}
	return nil
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestHandler(svc wishlist.Service) *wishlist.Handler {
	return wishlist.NewHandler(svc)
}

// ==================== CREATE TESTS ====================

func TestWishlistHandler_Create(t *testing.T) {
	t.Run("success_add_item", func(t *testing.T) {
		userID := uuid.New().String()
		productID := uuid.New().String()

		svc := &fakeWishlistService{
			createFunc: func(ctx context.Context, uid, pid string) (wishlist.AddItemResponse, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, productID, pid)

				return wishlist.AddItemResponse{
					Message: "Product added to wishlist successfully",
				}, nil
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + productID + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", userID)

		ctrl.Create(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Product added to wishlist successfully")
	})

	t.Run("error_user_not_authenticated", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "some-id"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		ctrl.Create(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error_invalid_json", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(`{invalid-json}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_missing_product_id", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{}`
		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_item_already_exists", func(t *testing.T) {
		svc := &fakeWishlistService{
			createFunc: func(ctx context.Context, userID, productID string) (wishlist.AddItemResponse, error) {
				return wishlist.AddItemResponse{}, wishlist.ErrItemAlreadyExists
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + uuid.New().String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Create(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("error_service_internal_error", func(t *testing.T) {
		svc := &fakeWishlistService{
			createFunc: func(ctx context.Context, userID, productID string) (wishlist.AddItemResponse, error) {
				return wishlist.AddItemResponse{}, errors.New("database error")
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + uuid.New().String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Create(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== LIST TESTS ====================

func TestWishlistHandler_List(t *testing.T) {
	t.Run("success_list_items", func(t *testing.T) {
		userID := uuid.New().String()
		wishlistID := uuid.New().String()

		svc := &fakeWishlistService{
			listFunc: func(ctx context.Context, uid string) (wishlist.WishlistResponse, error) {
				assert.Equal(t, userID, uid)

				return wishlist.WishlistResponse{
					ID:     wishlistID,
					UserID: userID,
					Items: []wishlist.WishlistItemResponse{
						{
							ID:        uuid.New().String(),
							ProductID: uuid.New().String(),
							Name:      "Product 1",
							Price:     100000.00,
							Stock:     10,
							ImageURL:  "image1.jpg",
							AddedAt:   time.Now(),
						},
						{
							ID:        uuid.New().String(),
							ProductID: uuid.New().String(),
							Name:      "Product 2",
							Price:     200000.00,
							Stock:     5,
							ImageURL:  "image2.jpg",
							AddedAt:   time.Now(),
						},
					},
					ItemCount: 2,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/wishlist", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Product 1")
		assert.Contains(t, w.Body.String(), "Product 2")
		assert.Contains(t, w.Body.String(), `"item_count":2`)
	})

	t.Run("success_empty_wishlist", func(t *testing.T) {
		userID := uuid.New().String()

		svc := &fakeWishlistService{
			listFunc: func(ctx context.Context, uid string) (wishlist.WishlistResponse, error) {
				return wishlist.WishlistResponse{
					UserID:    uid,
					Items:     []wishlist.WishlistItemResponse{},
					ItemCount: 0,
				}, nil
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/wishlist", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"item_count":0`)
	})

	t.Run("error_user_not_authenticated", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/wishlist", nil)

		ctrl.List(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error_service_error", func(t *testing.T) {
		svc := &fakeWishlistService{
			listFunc: func(ctx context.Context, userID string) (wishlist.WishlistResponse, error) {
				return wishlist.WishlistResponse{}, errors.New("database error")
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/wishlist", nil)
		c.Set("user_id", uuid.New().String())

		ctrl.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== DELETE TESTS ====================

func TestWishlistHandler_Delete(t *testing.T) {
	t.Run("success_delete_item", func(t *testing.T) {
		userID := uuid.New().String()
		productID := uuid.New().String()

		svc := &fakeWishlistService{
			deleteFunc: func(ctx context.Context, uid, pid string) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, productID, pid)
				return nil
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + productID + `"}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", userID)

		ctrl.Delete(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Product removed from wishlist successfully")
	})

	t.Run("error_user_not_authenticated", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "some-id"}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		ctrl.Delete(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error_invalid_json", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(`{invalid-json}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_missing_product_id", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_item_not_found", func(t *testing.T) {
		svc := &fakeWishlistService{
			deleteFunc: func(ctx context.Context, userID, productID string) error {
				return wishlist.ErrItemNotFound
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + uuid.New().String() + `"}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error_wishlist_not_found", func(t *testing.T) {
		svc := &fakeWishlistService{
			deleteFunc: func(ctx context.Context, userID, productID string) error {
				return wishlist.ErrWishlistNotFound
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + uuid.New().String() + `"}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error_service_internal_error", func(t *testing.T) {
		svc := &fakeWishlistService{
			deleteFunc: func(ctx context.Context, userID, productID string) error {
				return errors.New("database error")
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"product_id": "` + uuid.New().String() + `"}`
		c.Request = httptest.NewRequest(http.MethodDelete, "/wishlist", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
