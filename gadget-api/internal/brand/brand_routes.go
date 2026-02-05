package brand

import (
	"gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	brands := r.Group("/brands")
	{
		brands.GET("", handler.ListPublic)
		brands.GET("/:id", handler.GetByID)
	}

	adminBrands := r.Group("/admin/brands")
	adminBrands.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		adminBrands.GET("", handler.ListAdmin)
		adminBrands.POST("", handler.Create)
		adminBrands.PATCH("/:id", handler.Update)
		adminBrands.DELETE("/:id", handler.Delete)
		adminBrands.PATCH("/:id/restore", handler.Restore)
	}
}
