package review

import (
	"gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	reviews := r.Group("")
	reviews.Use(middleware.AuthMiddleware())
	{
		reviews.PATCH("/reviews/:id", handler.UpdateReview)
		reviews.DELETE("/reviews/:id", handler.DeleteReview)
		reviews.GET("/users/:userId/reviews", handler.GetReviewsByUserID)
	}
}
