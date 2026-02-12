package address_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-gadget-api/internal/address"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeAddressService struct {
	listFn      func(ctx context.Context, userID string) ([]address.AddressResponse, error)
	getByIDFn   func(ctx context.Context, addressID string, userID string) (address.AddressResponse, error)
	listAdminFn func(ctx context.Context, page, limit int) ([]address.AddressAdminResponse, int64, error)
	createFn    func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error)
	updateFn    func(ctx context.Context, id, userID string, req address.UpdateAddressRequest) (address.AddressResponse, error)
	deleteFn    func(ctx context.Context, id, userID string) error
}

func (f *fakeAddressService) List(ctx context.Context, userID string) ([]address.AddressResponse, error) {
	return f.listFn(ctx, userID)
}
func (f *fakeAddressService) GetByID(ctx context.Context, addressID string, userID string) (address.AddressResponse, error) {
	return f.getByIDFn(ctx, addressID, userID)
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

func TestAddressHandler_Create(t *testing.T) {
	userID := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		svc := &fakeAddressService{
			createFn: func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error) {
				assert.Equal(t, userID, req.UserID)
				return address.AddressResponse{Label: "Home"}, nil
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.POST("/addresses", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Create(c)
		})

		body := `{
			"label": "Home",
			"recipientName": "John Doe",
			"recipientPhone": "08123456789",
			"street": "Jl Test",
			"city": "Jakarta",
			"province": "DKI Jakarta",
			"postalCode": "12345",
			"isPrimary": true
		}`

		req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("InvalidBody", func(t *testing.T) {
		ctrl := address.NewHandler(&fakeAddressService{})
		router := setupTestRouter()

		router.POST("/addresses", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Create(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		svc := &fakeAddressService{
			createFn: func(ctx context.Context, req address.CreateAddressRequest) (address.AddressResponse, error) {
				return address.AddressResponse{}, errors.New("failed")
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.POST("/addresses", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Create(c)
		})

		body := `{
			"label": "Home",
			"recipientName": "John Doe",
			"recipientPhone": "08123456789",
			"street": "Jl Test",
			"city": "Jakarta",
			"province": "DKI Jakarta",
			"postalCode": "12345",
			"isPrimary": true
		}`
		req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAddressHandler_List(t *testing.T) {
	userID := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		svc := &fakeAddressService{
			listFn: func(ctx context.Context, uid string) ([]address.AddressResponse, error) {
				return []address.AddressResponse{{Label: "Home"}}, nil
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.GET("/addresses", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.List(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Failed", func(t *testing.T) {
		svc := &fakeAddressService{
			listFn: func(ctx context.Context, uid string) ([]address.AddressResponse, error) {
				return nil, errors.New("db error")
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.GET("/addresses", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.List(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/addresses", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAddressHandler_Detail(t *testing.T) {
	userID := uuid.New().String()
	addrID := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		svc := &fakeAddressService{
			getByIDFn: func(ctx context.Context, id, uid string) (address.AddressResponse, error) {
				assert.Equal(t, addrID, id)
				return address.AddressResponse{ID: addrID, Label: "Home"}, nil
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.GET("/addresses/:id", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Detail(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/addresses/"+addrID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		svc := &fakeAddressService{
			getByIDFn: func(ctx context.Context, id, uid string) (address.AddressResponse, error) {
				return address.AddressResponse{}, errors.New("not found")
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.GET("/addresses/:id", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Detail(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/addresses/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAddressHandler_Delete(t *testing.T) {
	userID := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		svc := &fakeAddressService{
			deleteFn: func(ctx context.Context, id, uID string) error {
				return nil
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.DELETE("/addresses/:id", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/addresses/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Failed", func(t *testing.T) {
		svc := &fakeAddressService{
			deleteFn: func(ctx context.Context, id, uID string) error {
				return errors.New("delete failed")
			},
		}

		router := setupTestRouter()
		ctrl := address.NewHandler(svc)

		router.DELETE("/addresses/:id", func(c *gin.Context) {
			c.Set("user_id_validated", userID)
			ctrl.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/addresses/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
