package customer_test

import (
	"context"
	"errors"
	"go-gadget-api/internal/customer"
	"go-gadget-api/internal/shared/database/helper"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeCustomerService struct {
	UpdateProfileFn func(ctx context.Context, userID string, req customer.UpdateProfileRequest) (customer.CustomerResponse, error)
}

func (f *fakeCustomerService) UpdateProfile(
	ctx context.Context,
	userID string,
	req customer.UpdateProfileRequest,
) (customer.CustomerResponse, error) {
	return f.UpdateProfileFn(ctx, userID, req)
}

func newTestCustomerHandler(svc customer.Service) *customer.Handler {
	return customer.NewHandler(svc)
}

func TestCustomerHandler_UpdateProfile(t *testing.T) {
	t.Run("success_update_profile", func(t *testing.T) {
		userID := "user-123"

		svc := &fakeCustomerService{
			UpdateProfileFn: func(ctx context.Context, uid string, req customer.UpdateProfileRequest) (customer.CustomerResponse, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, "New Name", helper.StringPtrValue(req.Name))

				return customer.CustomerResponse{
					ID:   userID,
					Name: "New Name",
				}, nil
			},
		}

		handler := newTestCustomerHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// body
		body := `{"name":"New Name"}`
		req := httptest.NewRequest(http.MethodPut, "/me", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		c.Request = req

		// context user
		c.Set("user_id", userID)
		c.Set("user_id_validated", userID)

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error_invalid_payload", func(t *testing.T) {
		userID := "user-123"
		svc := &fakeCustomerService{}
		handler := newTestCustomerHandler(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"name": "New Name"`
		req := httptest.NewRequest(http.MethodPut, "/me", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		c.Request = req
		c.Set("user_id", userID)
		c.Set("user_id_validated", userID)

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error_service_failure", func(t *testing.T) {
		userID := "user-123"

		// Simulasikan error dari layer service
		svc := &fakeCustomerService{
			UpdateProfileFn: func(ctx context.Context, uid string, req customer.UpdateProfileRequest) (customer.CustomerResponse, error) {
				return customer.CustomerResponse{}, errors.New("internal server error")
			},
		}

		handler := newTestCustomerHandler(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"name":"New Name"}`
		req := httptest.NewRequest(http.MethodPut, "/me", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		c.Request = req
		c.Set("user_id", userID)
		c.Set("user_id_validated", userID)

		handler.UpdateProfile(c)
		// ===== DEBUG LOG =====
		t.Log("status:", w.Code)
		t.Log("body:", w.Body.String())

		// Handler harus mengembalikan 500 jika service error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("error_unauthorized_no_user_id", func(t *testing.T) {
		svc := &fakeCustomerService{}
		handler := newTestCustomerHandler(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"name":"New Name"}`
		c.Request = httptest.NewRequest(http.MethodPut, "/me", strings.NewReader(body))

		handler.UpdateProfile(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
