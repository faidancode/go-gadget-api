package auth

import (
	"go-gadget-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	auth := r.Group("/auth")
	{
		// 1. Register (Sangat Ketat - per IP)
		// Mencegah bot membuat ribuan akun palsu.
		// limit 0.05 rps = 1 request per 20 detik.
		auth.POST("/register",
			middleware.RateLimitByIP(0.05, 1),
			handler.Register,
		)

		// 2. Login (Ketat - per IP)
		// Mencegah Brute Force password.
		// limit 0.1 rps = 1 request per 10 detik.
		auth.POST("/login",
			middleware.RateLimitByIP(0.1, 3),
			handler.Login,
		)

		// 3. Refresh Token (Menengah - per IP)
		// Biasanya dipanggil otomatis oleh frontend, beri sedikit kelonggaran.
		auth.POST("/refresh",
			middleware.RateLimitByIP(0.5, 2),
			handler.RefreshToken,
		)

		auth.POST("/password-reset/request",
			middleware.RateLimitByIP(0.2, 2),
			handler.RequestPasswordReset,
		)

		auth.POST("/password-reset/confirm",
			middleware.RateLimitByIP(0.2, 2),
			handler.ResetPassword,
		)

		auth.POST("/email-confirmation/request",
			middleware.RateLimitByIP(0.2, 2),
			handler.RequestEmailConfirmation,
		)

		auth.POST("/email-confirmation/resend",
			middleware.RateLimitByIP(0.2, 2),
			handler.ResendEmailConfirmation,
		)

		auth.GET("/email-confirmation/verify",
			middleware.RateLimitByIP(1, 3),
			handler.ConfirmEmailByToken,
		)

		auth.POST("/email-confirmation/verify-pin",
			middleware.RateLimitByIP(1, 3),
			handler.ConfirmEmailByPin,
		)

		// 4. Logout & Me (Authenticated - per User)
		// Menggunakan middleware AuthMiddleware dulu untuk mendapatkan user_id
		authenticated := auth.Group("/")
		authenticated.Use(middleware.AuthMiddleware())
		{
			// Info user cukup sering dipanggil saat app startup/refresh (Longgar)
			authenticated.GET("/me",
				middleware.RateLimitByUser(5, 10),
				handler.Me,
			)

			// Logout tidak perlu terlalu longgar
			authenticated.POST("/logout",
				middleware.RateLimitByUser(1, 2),
				handler.Logout,
			)
		}
	}
}
