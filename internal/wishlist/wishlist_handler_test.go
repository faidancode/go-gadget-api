package wishlist_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-gadget-api/internal/shared/database/helper"
	"go-gadget-api/internal/wishlist"
	"net/http"
	"net/http/httptest"
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

		requestBody := map[string]string{
			"productId": productID,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/wishlists/items", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		// -------------------------

		c.Set("user_id_validated", userID)

		ctrl.Create(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Product added to wishlist successfully")
	})

	t.Run("error_user_not_authenticated", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		productID := uuid.New().String()

		req := httptest.NewRequest(http.MethodPost, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}

		// tidak set user_id_validated

		ctrl.Create(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error_missing_product_id", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodPost, "/wishlists/items/", nil)
		c.Request = req

		c.Set("user_id_validated", uuid.New().String())
		// tidak set param

		ctrl.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_item_already_exists", func(t *testing.T) {
		productID := uuid.New().String()
		userID := uuid.New().String()

		svc := &fakeWishlistService{
			createFunc: func(ctx context.Context, uID, pID string) (wishlist.AddItemResponse, error) {
				// Memastikan data yang sampai ke service tetap benar
				assert.Equal(t, userID, uID)
				assert.Equal(t, productID, pID)
				return wishlist.AddItemResponse{}, wishlist.ErrItemAlreadyExists
			},
		}

		ctrl := newTestHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := map[string]string{
			"productId": productID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/wishlists/items", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		c.Set("user_id_validated", userID)

		ctrl.Create(c)

		// Pastikan status code sesuai dengan mapping ErrItemAlreadyExists (biasanya 409 Conflict)
		assert.Equal(t, http.StatusConflict, w.Code)

		// Opsional: cek apakah pesan errornya sesuai
		assert.Contains(t, w.Body.String(), "ALREADY_EXISTS")
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

		productID := uuid.New().String()

		reqBody := map[string]string{
			"productId": productID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/wishlists/items", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req
		c.Set("user_id_validated", uuid.New().String())

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
							ID: uuid.New().String(),
							Product: wishlist.WishlistProductResponse{
								ID:           uuid.New().String(),
								Slug:         "product-1",
								Name:         "Product 1",
								CategoryName: "Category A",
								Price:        100000, // cents
								Stock:        10,
								ImageURL:     helper.StringPtr("image1.jpg"),
							},
						},
						{
							ID: uuid.New().String(),
							Product: wishlist.WishlistProductResponse{
								ID:           uuid.New().String(),
								Slug:         "product-2",
								Name:         "Product 2",
								CategoryName: "Category B",
								Price:        200000,
								Stock:        5,
								ImageURL:     helper.StringPtr("image2.jpg"),
							},
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
		c.Set("user_id_validated", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Product 1")
		assert.Contains(t, w.Body.String(), "Product 2")
		assert.Contains(t, w.Body.String(), `"itemCount":2`)
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
		c.Set("user_id_validated", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"itemCount":0`)
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
		c.Set("user_id_validated", uuid.New().String())

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

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}
		c.Set("user_id_validated", userID)

		ctrl.Delete(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error_user_not_authenticated", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		productID := uuid.New().String()

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}

		// tidak set user_id_validated

		ctrl.Delete(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error_missing_product_id", func(t *testing.T) {
		ctrl := newTestHandler(&fakeWishlistService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/", nil)
		c.Request = req

		// tidak set param
		c.Set("user_id_validated", uuid.New().String())

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

		productID := uuid.New().String()

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}
		c.Set("user_id_validated", uuid.New().String())

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

		productID := uuid.New().String()

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}
		c.Set("user_id_validated", uuid.New().String())

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

		productID := uuid.New().String()

		req := httptest.NewRequest(http.MethodDelete, "/wishlists/items/"+productID, nil)
		c.Request = req

		c.Params = gin.Params{
			{Key: "productId", Value: productID},
		}
		c.Set("user_id_validated", uuid.New().String())

		ctrl.Delete(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
