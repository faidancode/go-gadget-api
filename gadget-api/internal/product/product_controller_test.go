package product_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"gadget-api/internal/product"
	producterrors "gadget-api/internal/product/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

//
// ==================== FAKE SERVICE ====================
//

type fakeProductService struct {
	CreateFn     func(ctx context.Context, req product.CreateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req product.UpdateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error)
	ListPublicFn func(ctx context.Context, req product.ListPublicRequest) ([]product.ProductPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, req product.ListProductAdminRequest) ([]product.ProductAdminResponse, int64, error)
	GetByIDFn    func(ctx context.Context, id string) (product.ProductAdminResponse, error)
	GetBySlugFn  func(ctx context.Context, slug string) (product.ProductDetailResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (product.ProductAdminResponse, error)
}

func (f *fakeProductService) Create(ctx context.Context, req product.CreateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error) {
	if f.CreateFn == nil {
		return product.ProductAdminResponse{}, nil
	}
	return f.CreateFn(ctx, req, file, filename)
}

func (f *fakeProductService) Update(ctx context.Context, id string, req product.UpdateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error) {
	if f.UpdateFn == nil {
		return product.ProductAdminResponse{}, nil
	}
	return f.UpdateFn(ctx, id, req, file, filename)
}

func (f *fakeProductService) ListPublic(ctx context.Context, req product.ListPublicRequest) ([]product.ProductPublicResponse, int64, error) {
	if f.ListPublicFn == nil {
		return nil, 0, nil
	}
	return f.ListPublicFn(ctx, req)
}

func (f *fakeProductService) ListAdmin(ctx context.Context, req product.ListProductAdminRequest) ([]product.ProductAdminResponse, int64, error) {
	if f.ListAdminFn == nil {
		return nil, 0, nil
	}
	return f.ListAdminFn(ctx, req)
}

func (f *fakeProductService) GetByID(ctx context.Context, id string) (product.ProductAdminResponse, error) {
	if f.GetByIDFn == nil {
		return product.ProductAdminResponse{}, nil
	}
	return f.GetByIDFn(ctx, id)
}

func (f *fakeProductService) GetBySlug(ctx context.Context, slug string) (product.ProductDetailResponse, error) {
	if f.GetBySlugFn == nil {
		return product.ProductDetailResponse{}, nil
	}
	return f.GetBySlugFn(ctx, slug)
}

func (f *fakeProductService) Delete(ctx context.Context, id string) error {
	if f.DeleteFn == nil {
		return nil
	}
	return f.DeleteFn(ctx, id)
}

func (f *fakeProductService) Restore(
	ctx context.Context,
	id string,
) (product.ProductAdminResponse, error) {
	if f.RestoreFn == nil {
		return product.ProductAdminResponse{}, nil
	}
	return f.RestoreFn(ctx, id)
}

//
// ==================== HELPERS ====================
//

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestHandler(svc product.Service) *product.Handler {
	return product.NewHandler(svc)
}

func createMultipartForm(fields map[string]string, fileField, filename string, content []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}

	if fileField != "" && filename != "" {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
			return nil, "", err
		}
	}

	ct := writer.FormDataContentType()
	_ = writer.Close()
	return body, ct, nil
}

//
// ==================== CREATE ====================
//

func TestCreateProduct(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeProductService{
			CreateFn: func(ctx context.Context, req product.CreateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error) {
				assert.Equal(t, "Product", req.Name)
				return product.ProductAdminResponse{ID: uuid.NewString()}, nil
			},
		}

		r := setupTestRouter()
		r.POST("/products", newTestHandler(svc).Create)

		body, ct, _ := createMultipartForm(
			map[string]string{
				"category_id": uuid.NewString(),
				"name":        "Product",
				"price":       "10000",
				"stock":       "5",
			},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPost, "/products", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeProductService{
			CreateFn: func(ctx context.Context, req product.CreateProductRequest, file multipart.File, filename string) (product.ProductAdminResponse, error) {
				return product.ProductAdminResponse{}, errors.New("db error")
			},
		}

		r := setupTestRouter()
		r.POST("/products", newTestHandler(svc).Create)

		body, ct, _ := createMultipartForm(
			map[string]string{
				"category_id": uuid.NewString(),
				"name":        "Product",
				"price":       "10000",
				"stock":       "5",
			},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPost, "/products", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

//
// ==================== LIST ADMIN ====================
//

func TestListAdminProducts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeProductService{
			ListAdminFn: func(ctx context.Context, req product.ListProductAdminRequest) ([]product.ProductAdminResponse, int64, error) {
				assert.Equal(t, 1, req.Page)
				assert.Equal(t, 10, req.Limit)
				return []product.ProductAdminResponse{
					{ID: uuid.NewString(), Name: "Product"},
				}, 1, nil
			},
		}

		r := setupTestRouter()
		r.GET("/admin/products", newTestHandler(svc).GetAdminList)

		req := httptest.NewRequest(http.MethodGet, "/admin/products?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeProductService{
			ListAdminFn: func(ctx context.Context, req product.ListProductAdminRequest) ([]product.ProductAdminResponse, int64, error) {
				return nil, 0, errors.New("db error")
			},
		}

		r := setupTestRouter()
		r.GET("/admin/products", newTestHandler(svc).GetAdminList)

		req := httptest.NewRequest(http.MethodGet, "/admin/products", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

//
// ==================== GET BY ID (ADMIN) ====================
//

func TestGetProductByID(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeProductService{
			GetByIDFn: func(ctx context.Context, pid string) (product.ProductAdminResponse, error) {
				assert.Equal(t, id, pid)
				return product.ProductAdminResponse{ID: pid}, nil
			},
		}

		r := setupTestRouter()
		r.GET("/admin/products/:id", newTestHandler(svc).GetByID)

		req := httptest.NewRequest(http.MethodGet, "/admin/products/"+id, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &fakeProductService{
			GetByIDFn: func(ctx context.Context, pid string) (product.ProductAdminResponse, error) {
				return product.ProductAdminResponse{}, producterrors.ErrProductNotFound
			},
		}

		r := setupTestRouter()
		r.GET("/admin/products/:id", newTestHandler(svc).GetByID)

		req := httptest.NewRequest(http.MethodGet, "/admin/products/"+id, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

//
// ==================== GET BY SLUG (PUBLIC) ====================
//

func TestGetProductBySlug(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeProductService{
			GetBySlugFn: func(ctx context.Context, slug string) (product.ProductDetailResponse, error) {
				assert.Equal(t, "iphone-15", slug)
				return product.ProductDetailResponse{Slug: slug}, nil
			},
		}

		r := setupTestRouter()
		r.GET("/products/:slug", newTestHandler(svc).GetBySlug)

		req := httptest.NewRequest(http.MethodGet, "/products/iphone-15", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &fakeProductService{
			GetBySlugFn: func(ctx context.Context, slug string) (product.ProductDetailResponse, error) {
				return product.ProductDetailResponse{}, producterrors.ErrProductNotFound
			},
		}

		r := setupTestRouter()
		r.GET("/products/:slug", newTestHandler(svc).GetBySlug)

		req := httptest.NewRequest(http.MethodGet, "/products/unknown", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

//
// ==================== DELETE ====================
//

func TestDeleteProduct(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeProductService{
			DeleteFn: func(ctx context.Context, pid string) error {
				assert.Equal(t, id, pid)
				return nil
			},
		}

		r := setupTestRouter()
		r.DELETE("/products/:id", newTestHandler(svc).Delete)

		req := httptest.NewRequest(http.MethodDelete, "/products/"+id, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		svc := &fakeProductService{
			DeleteFn: func(ctx context.Context, pid string) error {
				return producterrors.ErrProductNotFound
			},
		}

		r := setupTestRouter()
		r.DELETE("/products/:id", newTestHandler(svc).Delete)

		req := httptest.NewRequest(http.MethodDelete, "/products/"+id, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
