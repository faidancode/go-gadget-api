package middleware

import (
	"fmt"
	autherrors "go-gadget-api/internal/auth/errors"
	"go-gadget-api/internal/pkg/response"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get token
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			response.Error(c, autherrors.ErrUnauthorized.HTTPStatus, autherrors.ErrUnauthorized.Code, autherrors.ErrUnauthorized.Message, nil)
			c.Abort()
			return
		}

		// 2. Parse & Validate
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			errObj := autherrors.ErrInvalidToken
			if err != nil && strings.Contains(err.Error(), "expired") {
				errObj = autherrors.ErrTokenExpired
			}
			response.Error(c, errObj.HTTPStatus, errObj.Code, errObj.Message, nil)
			c.Abort()
			return
		}

		// 3. Extract Claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token claims", nil)
			c.Abort()
			return
		}

		// 4. Validate & Extract user_id
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "User ID not found in token", nil)
			c.Abort()
			return
		}

		role, _ := claims["role"].(string)

		// 5. Set validated values
		c.Set("user_id_validated", userID) // ✅ Langsung set validated
		c.Set("role", role)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil role dari context
		userRole, exists := c.Get("role")
		if !exists {
			response.Error(c, autherrors.ErrForbidden.HTTPStatus, autherrors.ErrForbidden.Code, autherrors.ErrForbidden.Message, nil)
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
			response.Error(c, autherrors.ErrForbidden.HTTPStatus, autherrors.ErrForbidden.Code, autherrors.ErrForbidden.Message, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
