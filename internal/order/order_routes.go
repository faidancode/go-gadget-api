package order

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler, rdb *redis.Client) {
	orders := r.Group("/orders")
	orders.Use(middleware.AuthMiddleware()) // Semua route order butuh login
	orders.Use(middleware.RateLimitByUser(5, 10))
	{
		// Customer Routes
		orders.POST("/checkout",
			middleware.RateLimitByUser(0.5, 1),
			middleware.Idempotency(rdb),
			handler.Checkout,
		)
		orders.GET("", handler.List)
		orders.GET("/:id", handler.Detail)
		orders.PATCH("/:id/cancel", handler.Cancel)
		orders.PATCH("/:id/status", handler.UpdateStatusByCustomer)

	}
	// Admin Routes (Management)
	adminOrders := r.Group("/admin/orders")
	adminOrders.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
	adminOrders.Use(middleware.RateLimitByIP(10, 20))
	{
		adminOrders.GET("", handler.ListAdmin)
		adminOrders.PATCH("/:id/status", handler.UpdateStatusByAdmin)
	}
}
