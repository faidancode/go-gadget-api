package cart

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	carts := r.Group("/carts")
	carts.Use(middleware.AuthMiddleware())
	{
		// 1. Ambil Data Cart (Detail & Count)
		// Biasanya dipanggil setiap kali pindah halaman atau update state di FE.
		// Diberi kelonggaran: 5 req/sec.
		carts.GET("/detail", middleware.RateLimitByUser(5, 10), handler.Detail)
		carts.GET("/count", middleware.RateLimitByUser(5, 10), handler.Count)

		// 2. Inisialisasi/Hapus Cart
		// Operasi ini cukup berat dan jarang dilakukan berturut-turut.
		// Dibatasi: 1 req/sec.
		carts.POST("", middleware.RateLimitByUser(1, 2), handler.Create)
		carts.DELETE("", middleware.RateLimitByUser(1, 2), handler.Delete)
		carts.DELETE("/clear", middleware.RateLimitByUser(1, 2), handler.ClearCart)

		// 3. Item Management (Sub-Group)
		// Operasi penambahan/pengurangan qty sangat rawan spamming.
		// Dibatasi: 2 req/sec (cukup untuk user yang mengklik cepat tombol +/-)
		items := carts.Group("/items/:productId")
		{
			itemMutationLimit := middleware.RateLimitByUser(2, 4)

			items.POST("", itemMutationLimit, handler.AddItem)
			items.PATCH("", itemMutationLimit, handler.UpdateQty)
			items.POST("/increment", itemMutationLimit, handler.Increment)
			items.POST("/decrement", itemMutationLimit, handler.Decrement)
			items.DELETE("", itemMutationLimit, handler.DeleteItem)
		}
	}
}
