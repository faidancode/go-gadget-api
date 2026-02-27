package address

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	address := r.Group("/addresses")
	address.Use(middleware.AuthMiddleware())
	{
		// 1. List & Detail Alamat
		// User biasanya mengakses ini saat checkout atau di halaman profil.
		// Limit 5 rps, burst 10 (cukup longgar untuk navigasi UI).
		address.GET("",
			middleware.RateLimitByUser(5, 10),
			handler.List,
		)
		address.GET("/:id",
			middleware.RateLimitByUser(5, 10),
			handler.Detail,
		)

		// 2. Create & Update Alamat
		// Menambah alamat biasanya melibatkan integrasi API pihak ketiga (misal: RajaOngkir/GoSend)
		// untuk validasi kodepos/koordinat. Kita perketat untuk menghemat kuota API pihak ketiga.
		// Limit 0.5 rps = 1 request per 2 detik.
		addressMutationLimit := middleware.RateLimitByUser(0.5, 2)

		address.POST("", addressMutationLimit, handler.Create)
		address.PUT("/:id", addressMutationLimit, handler.Update)

		// 3. Delete Alamat
		// Menghapus alamat bersifat destruktif.
		// Limit 1 rps, burst 2.
		address.DELETE("/:id",
			middleware.RateLimitByUser(1, 2),
			handler.Delete,
		)
	}
}
