package product

import (
	"gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	products := r.Group("/products")
	{
		products.GET("", handler.GetPublicList)
		products.GET("/:slug", handler.GetBySlug)
	}
	optional := products.Group("")
	optional.Use(middleware.OptionalAuthMiddleware())
	{
		optional.GET(
			"/:slug/reviews/eligibility",
			handler.CheckReviewEligibility,
		)
	}

	adminProducts := r.Group("/admin/products")
	adminProducts.Use(middleware.AuthMiddleware())
	adminProducts.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
	{
		adminProducts.GET("", handler.GetAdminList)
		adminProducts.POST("", handler.Create)
		adminProducts.PATCH("/:id", handler.Update)
		adminProducts.DELETE("/:id", handler.Delete)
		adminProducts.PATCH("/:id/restore", handler.Restore)
	}
}
