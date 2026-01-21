package category

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

type fakeCategoryService struct {
	CreateFn     func(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error)
	ListPublicFn func(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error)
	ListAdminFn  func(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error)
	GetByIDFn    func(ctx context.Context, id string) (CategoryAdminResponse, error)
	UpdateFn     func(ctx context.Context, id string, req CreateCategoryRequest) (CategoryAdminResponse, error)
	DeleteFn     func(ctx context.Context, id string) error
	RestoreFn    func(ctx context.Context, id string) (CategoryAdminResponse, error)
}

func (f *fakeCategoryService) Create(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error) {
	return f.CreateFn(ctx, req)
}
func (f *fakeCategoryService) ListPublic(ctx context.Context, p, l int) ([]CategoryPublicResponse, int64, error) {
	return f.ListPublicFn(ctx, p, l)
}
func (f *fakeCategoryService) ListAdmin(ctx context.Context, req ListCategoryRequest) ([]CategoryAdminResponse, int64, error) {
	return f.ListAdminFn(ctx, req)
}
func (f *fakeCategoryService) GetByID(ctx context.Context, id string) (CategoryAdminResponse, error) {
	return f.GetByIDFn(ctx, id)
}
func (f *fakeCategoryService) Update(ctx context.Context, id string, req CreateCategoryRequest) (CategoryAdminResponse, error) {
	return f.UpdateFn(ctx, id, req)
}
func (f *fakeCategoryService) Delete(ctx context.Context, id string) error {
	return f.DeleteFn(ctx, id)
}
func (f *fakeCategoryService) Restore(ctx context.Context, id string) (CategoryAdminResponse, error) {
	return f.RestoreFn(ctx, id)
}

func setupCategoryTest() (*gin.Engine, *fakeCategoryService) {
	gin.SetMode(gin.TestMode)

	svc := &fakeCategoryService{}
	ctrl := NewController(svc)

	r := gin.New()
	r.GET("/categories", ctrl.ListPublic)
	r.POST("/categories", ctrl.Create)
	r.GET("/categories/:id", ctrl.GetByID)
	r.PUT("/categories/:id", ctrl.Update)
	r.DELETE("/categories/:id", ctrl.Delete)
	r.POST("/categories/:id/restore", ctrl.Restore)

	return r, svc
}

func TestCategoryController_Create(t *testing.T) {
	router, svc := setupCategoryTest()

	payload := CreateCategoryRequest{Name: "Phone"}

	svc.CreateFn = func(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error) {
		return CategoryAdminResponse{ID: uuid.New().String(), Name: req.Name}, nil
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var res response.ApiEnvelope
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	assert.True(t, res.Success)
}

func TestCategoryController_Create_ValidationError(t *testing.T) {
	router, svc := setupCategoryTest() // Ambil svc dari setup

	// Inisialisasi mock agar tidak nil
	svc.CreateFn = func(ctx context.Context, req CreateCategoryRequest) (CategoryAdminResponse, error) {
		return CategoryAdminResponse{}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryController_Update(t *testing.T) {
	router, svc := setupCategoryTest()
	id := uuid.New().String()

	payload := CreateCategoryRequest{Name: "Updated"}

	t.Run("Success", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, id string, req CreateCategoryRequest) (CategoryAdminResponse, error) {
			return CategoryAdminResponse{ID: id, Name: req.Name}, nil
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/categories/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.UpdateFn = func(ctx context.Context, id string, req CreateCategoryRequest) (CategoryAdminResponse, error) {
			return CategoryAdminResponse{}, errors.New("update failed")
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/categories/"+id, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCategoryController_ListPublic(t *testing.T) {
	router, svc := setupCategoryTest()

	t.Run("Success", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error) {
			return []CategoryPublicResponse{
				{ID: uuid.New().String(), Name: "Phone"},
			}, 1, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/categories?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res response.ApiEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.True(t, res.Success)
		assert.NotNil(t, res.Meta)
	})

	t.Run("Internal Error", func(t *testing.T) {
		svc.ListPublicFn = func(ctx context.Context, page, limit int) ([]CategoryPublicResponse, int64, error) {
			return nil, 0, errors.New("db error")
		}

		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCategoryController_GetByID_NotFound(t *testing.T) {
	router, svc := setupCategoryTest()

	svc.GetByIDFn = func(ctx context.Context, id string) (CategoryAdminResponse, error) {
		return CategoryAdminResponse{}, errors.New("not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/categories/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCategoryController_Delete(t *testing.T) {
	router, svc := setupCategoryTest()
	id := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		svc.DeleteFn = func(ctx context.Context, id string) error {
			return nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/categories/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error", func(t *testing.T) {
		svc.DeleteFn = func(ctx context.Context, id string) error {
			return errors.New("delete failed")
		}

		req := httptest.NewRequest(http.MethodDelete, "/categories/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCategoryController_Restore(t *testing.T) {
	router, svc := setupCategoryTest()
	id := uuid.New().String()

	svc.RestoreFn = func(ctx context.Context, id string) (CategoryAdminResponse, error) {
		return CategoryAdminResponse{ID: id, Name: "Restored"}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/categories/"+id+"/restore", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
