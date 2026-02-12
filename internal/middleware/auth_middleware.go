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
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found {
			tokenString = ""
		}

		if tokenString == "" {
			if cookie, err := c.Cookie("access_token"); err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token not found", nil)
			c.Abort()
			return
		}

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

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token claims", nil)
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "User ID not found in token", nil)
			c.Abort()
			return
		}

		role, _ := claims["role"].(string)

		c.Set("user_id_validated", userID)
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
