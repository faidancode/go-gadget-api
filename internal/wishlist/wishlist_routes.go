package wishlist

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	wishlists := r.Group("wishlists")
	wishlists.Use(middleware.AuthMiddleware())
	{
		wishlists.GET("/items", handler.List)
		wishlists.POST("/items/:productId", handler.Create)
		wishlists.DELETE("/items/:productId", handler.Delete)
	}
}
