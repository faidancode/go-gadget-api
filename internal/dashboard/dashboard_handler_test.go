package dashboard_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-gadget-api/internal/dashboard"
	mock "go-gadget-api/internal/mock/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDashboardHandler_GetDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock.NewMockService(ctrl)
	h := dashboard.NewHandler(svc)
	r := gin.Default()

	r.GET("/api/v1/dashboard", h.GetDashboard)

	t.Run("success_get_dashboard", func(t *testing.T) {
		svc.EXPECT().
			GetDashboardData(gomock.Any()).
			Return(dashboard.DashboardResponse{
				Stats: dashboard.DashboardStatsResponse{
					TotalProducts: 10,
				},
				RecentOrders:         []dashboard.RecentOrderResponse{},
				CategoryDistribution: []dashboard.CategoryDistributionResponse{},
			}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res dashboard.DashboardResponse
		json.Unmarshal(resp.Body.Bytes(), &res)
		assert.Equal(t, int64(10), res.Stats.TotalProducts)
	})

	t.Run("error_service_failure", func(t *testing.T) {
		svc.EXPECT().
			GetDashboardData(gomock.Any()).
			Return(dashboard.DashboardResponse{}, errors.New("service error"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Contains(t, resp.Body.String(), "internal server error")
	})
}
