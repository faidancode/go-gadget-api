package address_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gadget-api/internal/address"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeAddressService struct {
	listFn      func(ctx context.Context, userID string) ([]address.AddressResponse, error)
	listAdminFn func(ctx context.Context, page, limit int) ([]address.AddressAdminResponse, int64, error)
	createFn    func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error)
	updateFn    func(ctx context.Context, id, userID string, req address.UpdateAddressRequest) (address.AddressResponse, error)
	deleteFn    func(ctx context.Context, id, userID string) error
}

func (f *fakeAddressService) List(ctx context.Context, userID string) ([]address.AddressResponse, error) {
	return f.listFn(ctx, userID)
}
func (f *fakeAddressService) ListAdmin(ctx context.Context, page, limit int) ([]address.AddressAdminResponse, int64, error) {
	return f.listAdminFn(ctx, page, limit)
}
func (f *fakeAddressService) Create(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error) {
	return f.createFn(ctx, req)
}
func (f *fakeAddressService) Update(ctx context.Context, id, userID string, req address.UpdateAddressRequest) (address.AddressResponse, error) {
	return f.updateFn(ctx, id, userID, req)
}
func (f *fakeAddressService) Delete(ctx context.Context, id, userID string) error {
	return f.deleteFn(ctx, id, userID)
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestHandler(svc address.Service) *address.Handler {
	return address.NewHandler(svc)
}

func TestAddressHandler_Create_Success(t *testing.T) {
	userID := uuid.New().String()

	svc := &fakeAddressService{
		createFn: func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error) {
			assert.Equal(t, userID, req.UserID)
			return address.AddressResponse{Label: "Home"}, nil
		},
	}

	router := setupTestRouter()
	ctrl := newTestHandler(svc)

	router.POST("/addresses", func(c *gin.Context) {
		c.Set("user_id", userID)
		ctrl.Create(c)
	})

	// ⬇️ LENGKAPKAN FIELD REQUIRED
	body := `{
		"label": "Home",
		"recipient_name": "John Doe",
		"recipient_phone": "08123456789",
		"street": "Jl Test",
		"city": "Jakarta",
		"province": "DKI Jakarta",
		"postal_code": "12345",
		"is_primary": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// ===== DEBUG LOG =====
	t.Log("status:", w.Code)
	t.Log("body:", w.Body.String())

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAddressHandler_Create_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	ctrl := newTestHandler(&fakeAddressService{})

	router.POST("/addresses", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		ctrl.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddressHandler_Create_ServiceError(t *testing.T) {
	userID := uuid.New().String()

	svc := &fakeAddressService{
		createFn: func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error) {
			return address.AddressResponse{}, errors.New("failed")
		},
	}

	router := setupTestRouter()
	ctrl := address.NewHandler(svc)

	router.POST("/addresses", func(c *gin.Context) {
		c.Set("user_id", userID)
		ctrl.Create(c)
	})

	// ⬇️ BODY HARUS VALID → supaya masuk ke SERVICE
	body := `{
		"label": "Home",
		"recipient_name": "John Doe",
		"recipient_phone": "08123456789",
		"street": "Jl Test",
		"city": "Jakarta",
		"province": "DKI Jakarta",
		"postal_code": "12345",
		"is_primary": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// ===== DEBUG LOG =====
	t.Log("status:", w.Code)
	t.Log("body:", w.Body.String())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddressHandler_List_Success(t *testing.T) {
	userID := uuid.New().String()

	svc := &fakeAddressService{
		listFn: func(ctx context.Context, uid string) ([]address.AddressResponse, error) {
			return []address.AddressResponse{{Label: "Home"}}, nil
		},
	}

	router := setupTestRouter()
	ctrl := newTestHandler(svc)

	router.GET("/addresses", func(c *gin.Context) {
		c.Set("user_id", userID)
		ctrl.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddressHandler_List_Failed(t *testing.T) {
	svc := &fakeAddressService{
		listFn: func(ctx context.Context, uid string) ([]address.AddressResponse, error) {
			return nil, errors.New("db error")
		},
	}

	router := setupTestRouter()
	ctrl := newTestHandler(svc)

	router.GET("/addresses", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		ctrl.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddressHandler_Delete_Success(t *testing.T) {
	svc := &fakeAddressService{
		deleteFn: func(ctx context.Context, id, userID string) error {
			return nil
		},
	}

	router := setupTestRouter()
	ctrl := newTestHandler(svc)

	router.DELETE("/addresses/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		ctrl.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/addresses/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddressHandler_Delete_Failed(t *testing.T) {
	svc := &fakeAddressService{
		deleteFn: func(ctx context.Context, id, userID string) error {
			return errors.New("delete failed")
		},
	}

	router := setupTestRouter()
	ctrl := newTestHandler(svc)

	router.DELETE("/addresses/:id", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		ctrl.Delete(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/addresses/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
