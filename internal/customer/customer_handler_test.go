package customer_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-gadget-api/internal/customer"
	mock "go-gadget-api/internal/mock/customer"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCustomerHandler_UpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := customer.NewHandler(svc)
	r := gin.Default()

	r.PATCH("/api/v1/customers/profile", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		h.UpdateProfile(c)
	})

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		svc.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(customer.CustomerResponse{
				ID:    userID.String(),
				Name:  "New Name",
				Email: "test@example.com",
			}, nil)

		body := map[string]string{"name": "New Name"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/customers/profile", bytes.NewBuffer(jsonBody))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res customer.CustomerResponse
		json.Unmarshal(resp.Body.Bytes(), &res)
		assert.Equal(t, "New Name", res.Name)
	})
}

func TestCustomerHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := customer.NewHandler(svc)
	r := gin.Default()

	r.GET("/api/v1/customers", h.List)

	t.Run("success", func(t *testing.T) {
		svc.EXPECT().
			ListCustomers(gomock.Any()).
			Return([]customer.CustomerListResponse{
				{ID: uuid.New().String(), Name: "Cust 1", Email: "c1@ex.com"},
			}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestCustomerHandler_ToggleStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := customer.NewHandler(svc)
	r := gin.Default()

	r.PATCH("/api/v1/customers/:id/status", h.ToggleStatus)

	t.Run("success_toggle_status", func(t *testing.T) {
		targetID := uuid.New().String()
		svc.EXPECT().
			ToggleCustomerStatus(gomock.Any(), targetID, false).
			Return(customer.CustomerListResponse{
				ID:       targetID,
				IsActive: false,
			}, nil)

		body := map[string]interface{}{"is_active": false}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/customers/"+targetID+"/status", bytes.NewBuffer(jsonBody))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("error_invalid_json_payload", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/customers/some-id/status", bytes.NewBuffer([]byte("invalid-json")))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		// bindJSON biasanya mengembalikan 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestCustomerHandler_GetDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := customer.NewHandler(svc)
	r := gin.Default()

	r.GET("/api/v1/customers/:id", h.GetDetails)

	t.Run("success_get_details", func(t *testing.T) {
		targetID := uuid.New().String()
		svc.EXPECT().
			GetCustomerDetails(gomock.Any(), targetID).
			Return(customer.CustomerDetailResponse{
				ID:        targetID,
				Name:      "John Doe",
				Addresses: []customer.AddressResponse{},
			}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers/"+targetID, nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "John Doe")
	})

	t.Run("error_not_found_from_service", func(t *testing.T) {
		targetID := uuid.New().String()
		// Simulasi error yang ditangani handleError
		svc.EXPECT().
			GetCustomerDetails(gomock.Any(), targetID).
			Return(customer.CustomerDetailResponse{}, errors.New("customer not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers/"+targetID, nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		// Tergantung implementasi h.handleError, biasanya 404 atau 500
		assert.NotEqual(t, http.StatusOK, resp.Code)
	})
}

func TestCustomerHandler_UpdateProfile_Negative(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := customer.NewHandler(svc)
	r := gin.Default()

	// Skenario Unauthorized (user_id tidak ada di context)
	r.PATCH("/api/v1/customers/profile/no-auth", h.UpdateProfile)

	// Skenario Service Error
	r.PATCH("/api/v1/customers/profile/error", func(c *gin.Context) {
		c.Set("user_id", "valid-user-id")
		h.UpdateProfile(c)
	})

	t.Run("error_unauthorized_no_context_id", func(t *testing.T) {
		body := map[string]string{"name": "New Name"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/customers/profile/no-auth", bytes.NewBuffer(jsonBody))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "unauthorized")
	})

	t.Run("error_service_failure", func(t *testing.T) {
		svc.EXPECT().
			UpdateProfile(gomock.Any(), "valid-user-id", gomock.Any()).
			Return(customer.CustomerResponse{}, errors.New("database connection failed"))

		body := map[string]string{"name": "New Name"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/customers/profile/error", bytes.NewBuffer(jsonBody))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.NotEqual(t, http.StatusOK, resp.Code)
	})
}
