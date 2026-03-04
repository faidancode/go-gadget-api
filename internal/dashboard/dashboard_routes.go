package dashboard

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	dashboardGroup := r.Group("/admin/dashboard")
	dashboardGroup.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
	)
	{
		// Query Dashboard biasanya melibatkan agregasi database yang berat (SUM, COUNT, dsb).
		// Batasi 2 rps dengan burst 5 agar tidak membebani DB jika direfresh terus-menerus.
		dashboardGroup.GET("",
			middleware.RateLimitByUser(2, 5),
			h.GetDashboard,
		)
	}
}
