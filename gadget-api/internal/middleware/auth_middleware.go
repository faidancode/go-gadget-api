package middleware

import (
	"fmt"
	"gadget-api/internal/pkg/response"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Ambil token dari cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication token", nil)
			c.Abort()
			return
		}

		// 2. Parse & Validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "Token is invalid or expired", nil)
			c.Abort()
			return
		}

		// 3. Set data user ke context jika diperlukan
		claims, _ := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])

		c.Next()
	}
}
