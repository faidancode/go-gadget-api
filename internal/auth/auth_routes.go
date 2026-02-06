package auth

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	auth := r.Group("/auth")
	{
		auth.GET("/me", middleware.AuthMiddleware(), handler.Me)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
		auth.POST("/logout", handler.Logout)
		auth.POST("/register", handler.Register)
	}
}
