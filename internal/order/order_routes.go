package order

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	orders := r.Group("/orders")
	orders.Use(middleware.AuthMiddleware()) // Semua route order butuh login
	{
		// Customer Routes
		orders.POST("/checkout", handler.Checkout)
		orders.GET("", handler.List)
		orders.GET("/:id", handler.Detail)
		orders.PATCH("/:id/cancel", handler.Cancel)
		orders.PATCH("/:id/status", handler.UpdateStatusByCustomer)

	}
	// Admin Routes (Management)
	adminOrders := r.Group("/admin/orders")
	adminOrders.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
	{
		adminOrders.GET("", handler.ListAdmin)
		adminOrders.PATCH("/:id/status", handler.UpdateStatusByAdmin)
	}
}
