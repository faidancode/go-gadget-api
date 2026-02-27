package auth

import (
	platform "go-gadget-api/internal/pkg/request"
	"go-gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	token, refreshToken, userResp, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "Email atau password salah", nil)
		return
	}
	isProd := os.Getenv("APP_ENV") == "production"

	if platform.IsWebClient(clientType) {
		// Set access_token cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "access_token",
			Value:    token,
			Path:     "/",
			MaxAge:   86400, // 1 hari
			HttpOnly: true,
			Secure:   isProd,
			SameSite: http.SameSiteLaxMode, // ✅ Explicit SameSite
		})

		// Set refresh_token cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   3600 * 24 * 7, // 7 hari
			HttpOnly: true,
			Secure:   isProd,
			SameSite: http.SameSiteLaxMode, // ✅ Explicit SameSite
		})
	}

	responseData := gin.H{
		"user":          userResp,
		"access_token":  token,
		"refresh_token": refreshToken,
	}

	response.Success(c, http.StatusOK, responseData, nil)
}

func (h *Handler) Me(c *gin.Context) {
	// asumsi middleware sudah set userID di context
	log.Printf("auth context: %+v\n", c.Keys)

	userID, ok := c.Get("user_id")
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	userResp, err := h.service.GetMe(
		c.Request.Context(),
		userID.(string),
	)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	response.Success(c, http.StatusOK, userResp, nil)
}

// auth/auth_Handler.go

func (h *Handler) Logout(c *gin.Context) {
	// Ambil isProd dari config
	isProd := os.Getenv("APP_ENV") == "production" // atau dari config Anda

	// Clear access_token
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode, // ✅ Harus sama dengan login
	})

	// Clear refresh_token
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode, // ✅ Harus sama dengan login
	})

	response.Success(c, http.StatusOK, "Logout success.", nil)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "REGISTER_FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	// 1. Deteksi Client
	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	var refreshToken string
	isWeb := platform.IsWebClient(clientType)

	// 2. Ambil Refresh Token (Cookie vs Body)
	if isWeb {
		var err error
		refreshToken, err = c.Cookie("refresh_token")
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "NO_REFRESH_TOKEN", "Missing refresh token", nil)
			return
		}
	} else {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required", nil)
			return
		}
		refreshToken = req.RefreshToken
	}

	// 3. Panggil Service untuk Verify & Issue New Tokens
	// Mengembalikan accessToken, newRefreshToken, userDetail, error
	newAccess, newRefresh, userResp, err := h.service.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error(), nil)
		return
	}

	isProd := os.Getenv("APP_ENV") == "production"

	// 4. Sinkronisasi Web (Set-Cookie)
	if isWeb {
		// Update Access Token di Cookie
		c.SetCookie("access_token", newAccess, 15*60, "/", "", isProd, true)
		// Update Refresh Token di Cookie
		c.SetCookie("refresh_token", newRefresh, 3600*24*7, "/", "", isProd, true)
	}

	// 5. Response Success (Tetap kirim body untuk sinkronisasi state di frontend)
	responseData := gin.H{
		"user":          userResp,
		"access_token":  newAccess,
		"refresh_token": newRefresh,
	}

	response.Success(c, http.StatusOK, responseData, nil)
}
