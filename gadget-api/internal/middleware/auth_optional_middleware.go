package middleware

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil token dari cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			// Tidak ada token → lanjut sebagai guest
			c.Next()
			return
		}

		// Parse JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			// Token ada tapi invalid / expired → tetap lanjut (anggap guest)
			c.Next()
			return
		}

		// Inject claims ke context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if role, exists := claims["role"]; exists {
				c.Set("role", role)
			}
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID)
			}
		}

		c.Next()
	}
}
