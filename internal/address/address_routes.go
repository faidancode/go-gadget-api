package address

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	address := r.Group("/address")
	address.Use(middleware.AuthMiddleware()) // Semua route order butuh login
	{
		address.POST("/", handler.Create)
		address.PUT("/:id", handler.Update)
		address.GET("", handler.List)
	}
}
