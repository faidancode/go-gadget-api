// internal/middleware/extract_user.go
package middleware

import (
	"go-gadget-api/internal/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExtractUserID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, exists := ctx.Get("user_id")
		if !exists {
			response.Error(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi", nil)
			ctx.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok || userIDStr == "" {
			response.Error(ctx, http.StatusUnauthorized, "INVALID_USER_ID", "Format user_id tidak valid", nil)
			ctx.Abort()
			return
		}

		// Set ulang dengan tipe yang sudah pasti string
		ctx.Set("user_id", userIDStr)
		ctx.Next()
	}
}
