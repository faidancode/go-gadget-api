package product

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	products := r.Group("/products")
	{
		// 1. Public List (Per IP)
		// Cukup longgar agar user asli nyaman browsing, tapi mencegah scraping masif.
		// limit 10 rps, burst 20
		products.GET("",
			middleware.RateLimitByIP(10, 20),
			handler.GetPublicList,
		)

		// 2. Detail Product (Per IP)
		// Sedikit lebih ketat dari list karena biasanya memicu query join yang lebih berat.
		products.GET("/:slug",
			middleware.RateLimitByIP(5, 10),
			handler.GetBySlug,
		)
	}

	// 3. Review Eligibility (Per User - Optional Auth)
	// Karena ini mengecek status belanja user, limitasi per User lebih tepat.
	optional := products.Group("")
	optional.Use(middleware.OptionalAuthMiddleware())
	{
		optional.GET(
			"/:slug/reviews/eligibility",
			middleware.RateLimitByUser(2, 5),
			handler.CheckReviewEligibility,
		)
	}

	// 4. Admin Product Routes (Per User/Admin)
	adminProducts := r.Group("/admin/products")
	adminProducts.Use(middleware.AuthMiddleware())
	adminProducts.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
	{
		// Dashboard Admin biasanya butuh rps lebih tinggi untuk operasional
		adminProducts.GET("",
			middleware.RateLimitByUser(10, 20),
			handler.GetAdminList,
		)

		adminProducts.GET("/:id",
			middleware.RateLimitByUser(10, 20),
			handler.GetByID,
		)

		// Create, Update, Delete, Restore (Ketat)
		// Mencegah ketidaksengajaan double-click atau script malfungsi yang merusak data.
		// limit 1 rps, burst 3
		adminMutationLimit := middleware.RateLimitByUser(1, 3)

		adminProducts.POST("", adminMutationLimit, handler.Create)
		adminProducts.PATCH("/:id", adminMutationLimit, handler.Update)
		adminProducts.DELETE("/:id", adminMutationLimit, handler.Delete)
		adminProducts.PATCH("/:id/restore", adminMutationLimit, handler.Restore)
	}
}
