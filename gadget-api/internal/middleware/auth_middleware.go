package middleware

import (
	"fmt"
	"gadget-api/internal/auth"
	"gadget-api/internal/pkg/response"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Ambil token dari cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			// Menggunakan ErrUnauthorized
			response.Error(c, auth.ErrUnauthorized.HTTPStatus, auth.ErrUnauthorized.Code, auth.ErrUnauthorized.Message, nil)
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
			// Cek jika error spesifik expired, jika tidak gunakan InvalidToken
			errObj := auth.ErrInvalidToken
			if strings.Contains(err.Error(), "expired") {
				errObj = auth.ErrTokenExpired
			}

			response.Error(c, errObj.HTTPStatus, errObj.Code, errObj.Message, nil)
			c.Abort()
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		c.Set("role", claims["role"])
		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil role dari context
		userRole, exists := c.Get("role")
		if !exists {
			response.Error(c, auth.ErrForbidden.HTTPStatus, auth.ErrForbidden.Code, auth.ErrForbidden.Message, nil)
			c.Abort()
			return
		}

		// Validasi role
		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			// Menggunakan ErrForbidden
			response.Error(c, auth.ErrForbidden.HTTPStatus, auth.ErrForbidden.Code, auth.ErrForbidden.Message, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
