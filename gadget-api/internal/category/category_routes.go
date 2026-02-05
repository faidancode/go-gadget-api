package category

import (
	"gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	categories := r.Group("/categories")
	{
		categories.GET("", handler.ListPublic)
		categories.GET("/:id", handler.GetByID)
	}

	adminCategories := r.Group("/admin/categories")
	adminCategories.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		adminCategories.GET("", handler.ListAdmin)
		adminCategories.POST("", handler.Create)
		adminCategories.PATCH("/:id", handler.Update)
		adminCategories.DELETE("/:id", handler.Delete)
		adminCategories.PATCH("/:id/restore", handler.Restore)
	}
}
