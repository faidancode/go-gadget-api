package brand_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"gadget-api/internal/brand"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeBrandService struct {
	// Mempertahankan ListPublicFn dan ListAdminFn sesuai permintaan
	CreateFn     func(ctx context.Context, req brand.CreateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error)
	ListPublicFn func(ctx context.Context, page, limit int) ([]brand.BrandPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, req brand.ListBrandRequest) ([]brand.BrandAdminResponse, int64, error)
	GetByIDFn    func(ctx context.Context, id string) (brand.BrandAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req brand.UpdateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (brand.BrandAdminResponse, error)
}

func (f *fakeBrandService) Create(ctx context.Context, req brand.CreateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error) {
	return f.CreateFn(ctx, req, file, filename)
}
func (f *fakeBrandService) ListPublic(ctx context.Context, p, l int) ([]brand.BrandPublicResponse, int64, error) {
	return f.ListPublicFn(ctx, p, l)
}
func (f *fakeBrandService) ListAdmin(ctx context.Context, req brand.ListBrandRequest) ([]brand.BrandAdminResponse, int64, error) {
	return f.ListAdminFn(ctx, req)
}
func (f *fakeBrandService) GetByID(ctx context.Context, id string) (brand.BrandAdminResponse, error) {
	return f.GetByIDFn(ctx, id)
}
func (f *fakeBrandService) Update(ctx context.Context, id string, req brand.UpdateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error) {
	return f.UpdateFn(ctx, id, req, file, filename)
}
func (f *fakeBrandService) Delete(ctx context.Context, id string) error {
	return f.DeleteFn(ctx, id)
}
func (f *fakeBrandService) Restore(ctx context.Context, id string) (brand.BrandAdminResponse, error) {
	return f.RestoreFn(ctx, id)
}

// ==================== HELPERS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
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

func TestCreateBrand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			CreateFn: func(ctx context.Context, req brand.CreateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error) {
				assert.Equal(t, "Apple", req.Name)
				return brand.BrandAdminResponse{ID: uuid.NewString(), Name: req.Name}, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.POST("/brands", ctrl.Create)

		// Menggunakan multipart form seperti di Product
		body, ct, _ := createMultipartForm(
			map[string]string{
				"name":        "Apple",
				"description": "Premium tech",
			},
			"image",
			"logo.png",
			[]byte("fake-image-content"),
		)

		req := httptest.NewRequest(http.MethodPost, "/brands", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("validation_error_missing_name", func(t *testing.T) {
		svc := &fakeBrandService{}
		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.POST("/brands", ctrl.Create)

		body, ct, _ := createMultipartForm(map[string]string{"description": "No Name"}, "", "", nil)

		req := httptest.NewRequest(http.MethodPost, "/brands", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeBrandService{
			CreateFn: func(ctx context.Context, req brand.CreateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error) {
				return brand.BrandAdminResponse{}, errors.New("create failed")
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.POST("/brands", ctrl.Create)

		body, ct, _ := createMultipartForm(
			map[string]string{"name": "Apple"},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPost, "/brands", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func TestUpdateBrand(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			UpdateFn: func(ctx context.Context, bid string, req brand.UpdateBrandRequest, file multipart.File, filename string) (brand.BrandAdminResponse, error) {
				assert.Equal(t, id, bid)
				return brand.BrandAdminResponse{ID: id, Name: req.Name}, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.PUT("/brands/:id", ctrl.Update)

		body, ct, _ := createMultipartForm(
			map[string]string{"name": "Updated Apple"},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPut, "/brands/"+id, body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeBrandService{}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.PUT("/brands/:id", ctrl.Update)

		body, ct, _ := createMultipartForm(map[string]string{"name": "X"}, "", "", nil)

		req := httptest.NewRequest(http.MethodPut, "/brands/invalid-uuid", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListPublicBrands(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			ListPublicFn: func(ctx context.Context, page, limit int) ([]brand.BrandPublicResponse, int64, error) {
				return []brand.BrandPublicResponse{{Name: "Apple"}}, 1, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/brands", ctrl.ListPublic)

		req := httptest.NewRequest(http.MethodGet, "/brands?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeBrandService{
			ListPublicFn: func(ctx context.Context, page, limit int) ([]brand.BrandPublicResponse, int64, error) {
				return nil, 0, errors.New("db error")
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/brands", ctrl.ListPublic)

		req := httptest.NewRequest(http.MethodGet, "/brands?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func TestBrandController_ListAdmin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			ListAdminFn: func(ctx context.Context, req brand.ListBrandRequest) ([]brand.BrandAdminResponse, int64, error) {
				assert.Equal(t, int32(1), req.Page)
				assert.Equal(t, int32(10), req.Limit)

				return []brand.BrandAdminResponse{
					{ID: uuid.NewString(), Name: "Apple"},
					{ID: uuid.NewString(), Name: "Samsung"},
				}, 2, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/admin/brands", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/brands?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success - default pagination", func(t *testing.T) {
		svc := &fakeBrandService{
			ListAdminFn: func(ctx context.Context, req brand.ListBrandRequest) ([]brand.BrandAdminResponse, int64, error) {
				assert.Equal(t, int32(1), req.Page)
				assert.Equal(t, int32(10), req.Limit)
				return []brand.BrandAdminResponse{}, 0, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/admin/brands", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/brands", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid query param", func(t *testing.T) {
		svc := &fakeBrandService{}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/admin/brands", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/brands?page=abc&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeBrandService{
			ListAdminFn: func(ctx context.Context, req brand.ListBrandRequest) ([]brand.BrandAdminResponse, int64, error) {
				return nil, 0, errors.New("service error")
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/admin/brands", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/brands?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetBrandByID(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			GetByIDFn: func(ctx context.Context, bid string) (brand.BrandAdminResponse, error) {
				assert.Equal(t, id, bid)
				return brand.BrandAdminResponse{ID: id, Name: "Apple"}, nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/brands/:id", ctrl.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/brands/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeBrandService{}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.GET("/brands/:id", ctrl.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/brands/invalid-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteBrand(t *testing.T) {
	id := uuid.NewString()
	t.Run("success", func(t *testing.T) {
		svc := &fakeBrandService{
			DeleteFn: func(ctx context.Context, bid string) error {
				assert.Equal(t, id, bid)
				return nil
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.DELETE("/brands/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/brands/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeBrandService{}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.DELETE("/brands/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/brands/invalid-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeBrandService{
			DeleteFn: func(ctx context.Context, id string) error {
				return errors.New("delete failed")
			},
		}

		r := setupTestRouter()
		ctrl := brand.NewController(svc)
		r.DELETE("/brands/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/brands/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}
