package cart

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	carts := r.Group("/carts")
	carts.Use(middleware.AuthMiddleware())
	// carts.Use(middleware.ExtractUserID())
	{
		carts.POST("", handler.Create)
		carts.GET("/detail", handler.Detail)
		carts.GET("/count", handler.Count)
		carts.DELETE("", handler.Delete)

		items := carts.Group("/items/:productId")
		{
			items.POST("", handler.AddItem)
			items.PATCH("", handler.UpdateQty)
			items.POST("/increment", handler.Increment)
			items.POST("/decrement", handler.Decrement)
			items.DELETE("", handler.DeleteItem)
		}
	}
}
