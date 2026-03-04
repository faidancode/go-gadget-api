package customer

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	customerGroup := r.Group("customers")
	customerGroup.Use(
		middleware.AuthMiddleware(),
	)
	{
		// Update profile biasanya jarang dilakukan.
		// 1 rps, burst 3 untuk mencegah spam update.
		customerGroup.PATCH("/profile",
			middleware.RateLimitByUser(1, 3),
			h.UpdateProfile,
		)
	}

	adminCustomerGroup := r.Group("admin/customers")
	adminCustomerGroup.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		// List customer: data bisa banyak, batasi agar tidak scraping.
		adminCustomerGroup.GET("",
			middleware.RateLimitByUser(5, 10),
			h.List,
		)

		adminCustomerGroup.GET("/:id",
			middleware.RateLimitByUser(10, 20),
			h.GetDetails,
		)

		adminCustomerGroup.GET("/:id/addresses",
			middleware.RateLimitByUser(10, 20),
			h.GetAddresses,
		)

		adminCustomerGroup.GET("/:id/orders",
			middleware.RateLimitByUser(10, 20),
			h.GetOrders,
		)

		// Toggle status: Operasi kritikal, batasi ketat.
		adminCustomerGroup.PATCH("/:id/status",
			middleware.RateLimitByUser(1, 2),
			h.ToggleStatus,
		)
	}
}
