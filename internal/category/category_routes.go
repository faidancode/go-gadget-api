package category

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	categories := r.Group("/categories")
	{
		// 1. List Public Categories
		// Karena data kategori jarang berubah dan sering di-cache,
		// kita beri limit longgar: 10 rps, burst 20.
		categories.GET("",
			middleware.RateLimitByIP(10, 20),
			handler.ListPublic,
		)

		// 2. Get Detail Category (Includ. List Product per Category)
		// Endpoint ini biasanya menarik data produk dalam jumlah banyak.
		// Limit 5 rps, burst 10.
		categories.GET("/:id",
			middleware.RateLimitByIP(5, 10),
			handler.GetByID,
		)
	}

	// 3. Admin Categories (Management)
	adminCategories := r.Group("/admin/categories")
	adminCategories.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		// Admin List (Longgar)
		adminCategories.GET("",
			middleware.RateLimitByUser(10, 20),
			handler.ListAdmin,
		)

		// Create, Update, Delete, Restore (Ketat)
		// Perubahan kategori berdampak pada banyak produk sekaligus.
		// Limit 1 rps, burst 3.
		categoryMutationLimit := middleware.RateLimitByUser(1, 3)

		adminCategories.POST("", categoryMutationLimit, handler.Create)
		adminCategories.PATCH("/:id", categoryMutationLimit, handler.Update)
		adminCategories.DELETE("/:id", categoryMutationLimit, handler.Delete)
		adminCategories.PATCH("/:id/restore", categoryMutationLimit, handler.Restore)
	}
}
