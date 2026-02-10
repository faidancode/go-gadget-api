package review

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	reviews := r.Group("reviews")
	reviews.Use(middleware.AuthMiddleware())
	{
		reviews.GET("", handler.GetReviewsByUserID)
		reviews.PATCH("/:id", handler.UpdateReview)
		reviews.DELETE("/:id", handler.DeleteReview)
	}
}
