package address

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	address := r.Group("/address")
	address.Use(middleware.AuthMiddleware()) // Semua route order butuh login
	{
		address.GET("", handler.List)
		address.POST("/", handler.Create)
		address.PUT("/:id", handler.Update)
	}
}
