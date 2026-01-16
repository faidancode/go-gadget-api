package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"gadget-api/internal/pkg/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

/*
====================================================
FAKE SERVICE (UNTUK CONTROLLER TEST)
====================================================
*/
type fakeProductService struct {
	ListPublicFn func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error)
	CreateFn     func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error)
	GetByIDFn    func(ctx context.Context, id string) (ProductAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req UpdateProductRequest) (ProductAdminResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (ProductAdminResponse, error)
}

func (f *fakeProductService) ListPublic(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
	return f.ListPublicFn(ctx, req)
}
func (f *fakeProductService) ListAdmin(ctx context.Context, p, l int, s, c, cid string) ([]ProductAdminResponse, int64, error) {
	return f.ListAdminFn(ctx, p, l, s, c, cid)
}
func (f *fakeProductService) Create(ctx context.Context, r CreateProductRequest) (ProductAdminResponse, error) {
	return f.CreateFn(ctx, r)
}
func (f *fakeProductService) GetByIDAdmin(ctx context.Context, id string) (ProductAdminResponse, error) {
	return f.GetByIDFn(ctx, id)
}
func (f *fakeProductService) Update(ctx context.Context, id string, r UpdateProductRequest) (ProductAdminResponse, error) {
	return f.UpdateFn(ctx, id, r)
}
func (f *fakeProductService) Delete(ctx context.Context, id string) error {
	return f.DeleteFn(ctx, id)
}
func (f *fakeProductService) Restore(ctx context.Context, id string) (ProductAdminResponse, error) {
	return f.RestoreFn(ctx, id)
}

/*
====================================================
SETUP ROUTER
====================================================
*/
func setupTest() (*gin.Engine, *fakeProductService) {
	gin.SetMode(gin.TestMode)

	svc := &fakeProductService{}
	ctrl := NewController(svc)

	r := gin.New()
	r.GET("/products", ctrl.GetPublicList)
	r.GET("/products/admin", ctrl.GetAdminList)
	r.POST("/products", ctrl.Create)
	r.GET("/products/:id", ctrl.GetByID)
	r.PUT("/products/:id", ctrl.Update)
	r.DELETE("/products/:id", ctrl.Delete)
	r.POST("/products/:id/restore", ctrl.Restore)

	return r, svc
}

/*
====================================================
GET PUBLIC LIST
====================================================
*/
func TestGetPublicList(t *testing.T) {
	router, svc := setupTest()

	t.Run("Success", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
			return []ProductPublicResponse{}, 10, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/products?page=1&limit=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res response.ApiEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.True(t, res.Success)
		assert.NotNil(t, res.Meta)
	})

	t.Run("Internal Error", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, req ListPublicRequest) ([]ProductPublicResponse, int64, error) {
			return nil, 0, errors.New("db error")
		}

		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

/*
====================================================
GET ADMIN LIST
====================================================
*/
func TestGetAdminList(t *testing.T) {
	router, svc := setupTest()

	svc.ListAdminFn = func(ctx context.Context, page, limit int, search, sortCol, categoryID string) ([]ProductAdminResponse, int64, error) {
		return []ProductAdminResponse{}, 0, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/products/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

/*
====================================================
CREATE PRODUCT
====================================================
*/
func TestCreateProduct(t *testing.T) {
	router, svc := setupTest()

	payload := CreateProductRequest{
		Name:       "Macbook",
		Price:      20000000,
		Stock:      10,
		CategoryID: uuid.New().String(),
	}

	t.Run("Success", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.CreateFn = func(ctx context.Context, req CreateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("create failed")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

/*
====================================================
GET BY ID
====================================================
*/
func TestGetProductByID(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	t.Run("Found", func(t *testing.T) {
		svc.GetByIDFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
			return ProductAdminResponse{ID: pid}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc.GetByIDFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("not found")
		}

		req := httptest.NewRequest(http.MethodGet, "/products/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

/*
====================================================
UPDATE PRODUCT
====================================================
*/
func TestUpdateProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	payload := UpdateProductRequest{
		Name:  "Updated",
		Price: 9999,
	}

	t.Run("Success", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, pid string, req UpdateProductRequest) (ProductAdminResponse, error) {
			return ProductAdminResponse{}, errors.New("product not found")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

/*
====================================================
DELETE PRODUCT
====================================================
*/
func TestDeleteProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	svc.DeleteFn = func(ctx context.Context, pid string) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/products/"+id, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

/*
====================================================
RESTORE PRODUCT
====================================================
*/
func TestRestoreProduct(t *testing.T) {
	router, svc := setupTest()
	id := uuid.New().String()

	svc.RestoreFn = func(ctx context.Context, pid string) (ProductAdminResponse, error) {
		return ProductAdminResponse{ID: pid}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/products/"+id+"/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
