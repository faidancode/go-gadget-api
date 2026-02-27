package review

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	reviews := r.Group("reviews")
	reviews.Use(middleware.AuthMiddleware())
	{
		// 1. Create Review (Sangat Ketat)
		// Menulis review butuh waktu untuk mengetik.
		// Limit 0.05 rps = 1 request per 20 detik untuk mencegah spam bot.
		reviews.POST("",
			middleware.RateLimitByUser(0.05, 1),
			handler.Create,
		)

		// 2. Get My Reviews (Normal)
		// User melihat daftar review yang pernah mereka buat.
		// Diberi limit 3 rps, burst 5.
		reviews.GET("",
			middleware.RateLimitByUser(3, 5),
			handler.GetReviewsByUserID,
		)

		// 3. Update & Delete Review (Ketat)
		// Perubahan ulasan tidak terjadi setiap saat.
		// Limit 0.2 rps = 1 request per 5 detik.
		reviewMutationLimit := middleware.RateLimitByUser(0.2, 2)

		reviews.PATCH("/:id", reviewMutationLimit, handler.UpdateReview)
		reviews.DELETE("/:id", reviewMutationLimit, handler.DeleteReview)
	}
}
