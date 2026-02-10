package customer

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	customers := r.Group("/customers")
	customers.Use(middleware.AuthMiddleware())
	{
		customers.PATCH("/profile", handler.UpdateProfile)
	}
}
