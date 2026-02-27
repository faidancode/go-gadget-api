package wishlist

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	wishlists := r.Group("wishlists")
	wishlists.Use(middleware.AuthMiddleware())
	{
		// 1. Get Wishlist List (Normal)
		// User melihat daftar barang impian mereka.
		// Limit 5 rps, burst 10 (cukup untuk navigasi antar halaman wishlist).
		wishlists.GET("/items",
			middleware.RateLimitByUser(5, 10),
			handler.List,
		)

		// 2. Add & Remove Item (Ketat)
		// Operasi ini melibatkan pengecekan relasi di database.
		// Limit 1 rps, burst 3.
		// Memberikan perlindungan dari bot atau ketidaksengajaan klik berulang.
		itemActionLimit := middleware.RateLimitByUser(1, 3)

		wishlists.POST("/items/:productId",
			itemActionLimit,
			handler.Create,
		)

		wishlists.DELETE("/items/:productId",
			itemActionLimit,
			handler.Delete,
		)
	}
}
