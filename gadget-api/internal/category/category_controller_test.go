package category_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"gadget-api/internal/category"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeCategoryService struct {
	// Mempertahankan ListPublicFn dan ListAdminFn sesuai permintaan
	CreateFn     func(ctx context.Context, req category.CreateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error)
	ListPublicFn func(ctx context.Context, page, limit int) ([]category.CategoryPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, req category.ListCategoryRequest) ([]category.CategoryAdminResponse, int64, error)
	GetByIDFn    func(ctx context.Context, id string) (category.CategoryAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req category.UpdateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (category.CategoryAdminResponse, error)
}

func (f *fakeCategoryService) Create(ctx context.Context, req category.CreateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error) {
	return f.CreateFn(ctx, req, file, filename)
}
func (f *fakeCategoryService) ListPublic(ctx context.Context, p, l int) ([]category.CategoryPublicResponse, int64, error) {
	return f.ListPublicFn(ctx, p, l)
}
func (f *fakeCategoryService) ListAdmin(ctx context.Context, req category.ListCategoryRequest) ([]category.CategoryAdminResponse, int64, error) {
	return f.ListAdminFn(ctx, req)
}
func (f *fakeCategoryService) GetByID(ctx context.Context, id string) (category.CategoryAdminResponse, error) {
	return f.GetByIDFn(ctx, id)
}
func (f *fakeCategoryService) Update(ctx context.Context, id string, req category.UpdateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error) {
	return f.UpdateFn(ctx, id, req, file, filename)
}
func (f *fakeCategoryService) Delete(ctx context.Context, id string) error {
	return f.DeleteFn(ctx, id)
}
func (f *fakeCategoryService) Restore(ctx context.Context, id string) (category.CategoryAdminResponse, error) {
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

func TestCreateCategory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			CreateFn: func(ctx context.Context, req category.CreateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error) {
				assert.Equal(t, "Apple", req.Name)
				return category.CategoryAdminResponse{ID: uuid.NewString(), Name: req.Name}, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.POST("/categorys", ctrl.Create)

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

		req := httptest.NewRequest(http.MethodPost, "/categorys", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("validation_error_missing_name", func(t *testing.T) {
		svc := &fakeCategoryService{}
		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.POST("/categorys", ctrl.Create)

		body, ct, _ := createMultipartForm(map[string]string{"description": "No Name"}, "", "", nil)

		req := httptest.NewRequest(http.MethodPost, "/categorys", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeCategoryService{
			CreateFn: func(ctx context.Context, req category.CreateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error) {
				return category.CategoryAdminResponse{}, errors.New("create failed")
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.POST("/categorys", ctrl.Create)

		body, ct, _ := createMultipartForm(
			map[string]string{"name": "Apple"},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPost, "/categorys", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func TestUpdateCategory(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			UpdateFn: func(ctx context.Context, bid string, req category.UpdateCategoryRequest, file multipart.File, filename string) (category.CategoryAdminResponse, error) {
				assert.Equal(t, id, bid)
				return category.CategoryAdminResponse{ID: id, Name: req.Name}, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.PUT("/categorys/:id", ctrl.Update)

		body, ct, _ := createMultipartForm(
			map[string]string{"name": "Updated Apple"},
			"",
			"",
			nil,
		)

		req := httptest.NewRequest(http.MethodPut, "/categorys/"+id, body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeCategoryService{}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.PUT("/categorys/:id", ctrl.Update)

		body, ct, _ := createMultipartForm(map[string]string{"name": "X"}, "", "", nil)

		req := httptest.NewRequest(http.MethodPut, "/categorys/invalid-uuid", body)
		req.Header.Set("Content-Type", ct)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListPublicCategorys(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			ListPublicFn: func(ctx context.Context, page, limit int) ([]category.CategoryPublicResponse, int64, error) {
				return []category.CategoryPublicResponse{{Name: "Apple"}}, 1, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/categorys", ctrl.ListPublic)

		req := httptest.NewRequest(http.MethodGet, "/categorys?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeCategoryService{
			ListPublicFn: func(ctx context.Context, page, limit int) ([]category.CategoryPublicResponse, int64, error) {
				return nil, 0, errors.New("db error")
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/categorys", ctrl.ListPublic)

		req := httptest.NewRequest(http.MethodGet, "/categorys?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func TestCategoryHandler_ListAdmin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			ListAdminFn: func(ctx context.Context, req category.ListCategoryRequest) ([]category.CategoryAdminResponse, int64, error) {
				assert.Equal(t, int32(1), req.Page)
				assert.Equal(t, int32(10), req.Limit)

				return []category.CategoryAdminResponse{
					{ID: uuid.NewString(), Name: "Apple"},
					{ID: uuid.NewString(), Name: "Samsung"},
				}, 2, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/admin/categorys", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/categorys?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success - default pagination", func(t *testing.T) {
		svc := &fakeCategoryService{
			ListAdminFn: func(ctx context.Context, req category.ListCategoryRequest) ([]category.CategoryAdminResponse, int64, error) {
				assert.Equal(t, int32(1), req.Page)
				assert.Equal(t, int32(10), req.Limit)
				return []category.CategoryAdminResponse{}, 0, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/admin/categorys", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/categorys", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid query param", func(t *testing.T) {
		svc := &fakeCategoryService{}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/admin/categorys", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/categorys?page=abc&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeCategoryService{
			ListAdminFn: func(ctx context.Context, req category.ListCategoryRequest) ([]category.CategoryAdminResponse, int64, error) {
				return nil, 0, errors.New("service error")
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/admin/categorys", ctrl.ListAdmin)

		req := httptest.NewRequest(http.MethodGet, "/admin/categorys?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetCategoryByID(t *testing.T) {
	id := uuid.NewString()

	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			GetByIDFn: func(ctx context.Context, bid string) (category.CategoryAdminResponse, error) {
				assert.Equal(t, id, bid)
				return category.CategoryAdminResponse{ID: id, Name: "Apple"}, nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/categorys/:id", ctrl.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/categorys/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeCategoryService{}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.GET("/categorys/:id", ctrl.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/categorys/invalid-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteCategory(t *testing.T) {
	id := uuid.NewString()
	t.Run("success", func(t *testing.T) {
		svc := &fakeCategoryService{
			DeleteFn: func(ctx context.Context, bid string) error {
				assert.Equal(t, id, bid)
				return nil
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.DELETE("/categorys/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/categorys/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("negative - invalid uuid", func(t *testing.T) {
		svc := &fakeCategoryService{}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.DELETE("/categorys/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/categorys/invalid-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative - service error", func(t *testing.T) {
		svc := &fakeCategoryService{
			DeleteFn: func(ctx context.Context, id string) error {
				return errors.New("delete failed")
			},
		}

		r := setupTestRouter()
		ctrl := category.NewHandler(svc)
		r.DELETE("/categorys/:id", ctrl.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/categorys/"+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

}
