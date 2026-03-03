package dashboard

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	dashboardGroup := r.Group("/admin/dashboard")
	{
		dashboardGroup.GET("", h.GetDashboard)
	}
}
