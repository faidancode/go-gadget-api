package order

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler, rdb *redis.Client) {
	// Group utama Order (User Side)
	orders := r.Group("/orders")
	orders.Use(middleware.AuthMiddleware())

	// Global limit untuk user agar tidak melakukan crawling data order mereka sendiri secara berlebihan
	// 5 req/sec dengan burst 10 (cukup longgar untuk browsing list)
	orders.Use(middleware.RateLimitByUser(5, 10))
	{
		// 1. Checkout (Sangat Ketat)
		// limit 0.1 rps = 1 request per 10 detik.
		// Sangat penting untuk mencegah double order accidental atau bot spam.
		orders.POST("/checkout",
			middleware.RateLimitByUser(0.1, 1),
			middleware.Idempotency(rdb),
			handler.Checkout,
		)

		// 2. List & Detail (Normal)
		// Mengikuti global limit (5 rps) sudah cukup aman.
		orders.GET("", handler.List)
		orders.GET("/:id", handler.Detail)

		// 3. Cancel & Complete (Menengah)
		// User tidak seharusnya membatalkan/menyelesaikan order berkali-kali dalam sekejap.
		// limit 0.5 rps = 1 request per 2 detik.
		orders.PATCH("/:id/cancel",
			middleware.RateLimitByUser(0.5, 2),
			handler.Cancel,
		)
		orders.PATCH("/:id/complete",
			middleware.RateLimitByUser(0.5, 2),
			handler.Complete,
		)
	}

	// Admin Routes (Management)
	adminOrders := r.Group("/admin/orders")
	adminOrders.Use(middleware.AuthMiddleware())
	adminOrders.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))

	// Admin biasanya butuh limit lebih tinggi karena melakukan monitoring/dashboard.
	// Menggunakan IP limit sebagai tambahan lapisan keamanan infrastruktur.
	adminOrders.Use(middleware.RateLimitByIP(10, 20))
	{
		adminOrders.GET("", handler.ListAdmin)

		// Update status order oleh admin
		// limit 2 rps untuk mencegah perubahan status massal yang tidak sengaja via script.
		adminOrders.PATCH("/:id/status",
			middleware.RateLimitByUser(2, 5),
			handler.UpdateStatusByAdmin,
		)
	}
}
