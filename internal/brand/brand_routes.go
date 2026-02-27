package brand

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	brands := r.Group("/brands")
	{
		// 1. List Public Brands
		// Data brand biasanya statis dan ringan.
		// Limit 10 rps, burst 20 per IP.
		brands.GET("",
			middleware.RateLimitByIP(10, 20),
			handler.ListPublic,
		)

		// 2. Get Detail Brand & Products
		// Sering digunakan untuk halaman katalog per brand.
		// Limit 5 rps, burst 10 per IP karena query produk lebih berat.
		brands.GET("/:id",
			middleware.RateLimitByIP(5, 10),
			handler.GetByID,
		)
	}

	// 3. Admin Brands (Management)
	adminBrands := r.Group("/admin/brands")
	adminBrands.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		// Admin List (Longgar)
		adminBrands.GET("",
			middleware.RateLimitByUser(10, 20),
			handler.ListAdmin,
		)

		// Write Operations (Ketat)
		// Limit 1 rps, burst 3 untuk mencegah duplikasi brand akibat lag atau double-click.
		brandMutationLimit := middleware.RateLimitByUser(1, 3)

		adminBrands.POST("", brandMutationLimit, handler.Create)
		adminBrands.PATCH("/:id", brandMutationLimit, handler.Update)
		adminBrands.DELETE("/:id", brandMutationLimit, handler.Delete)
		adminBrands.PATCH("/:id/restore", brandMutationLimit, handler.Restore)
	}
}
