package customer

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	customerGroup := r.Group("/customers")
	{
		// Customer self profile
		customerGroup.PATCH("/profile", h.UpdateProfile)

		// Admin only (assuming these should be protected)
		customerGroup.GET("", h.List)
		customerGroup.GET("/:id", h.GetDetails)
		customerGroup.PATCH("/:id/status", h.ToggleStatus)
	}
}
