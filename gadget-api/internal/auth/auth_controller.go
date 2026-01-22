package auth

import (
	platform "gadget-api/internal/pkg/request"
	"gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(s *Service) *Controller {
	return &Controller{service: s}
}

func (ctrl *Controller) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	token, refreshToken, userResp, err := ctrl.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "Email atau password salah", nil)
		return
	}
	isProd := os.Getenv("APP_ENV") == "production"

	if platform.IsWebClient(clientType) {
		c.SetCookie(
			"access_token",
			token,
			86400,
			"/",
			"",
			isProd,
			true,
		)

		c.SetCookie(
			"refresh_token",
			refreshToken,
			3600*24*7,
			"/",
			"",
			isProd,
			true)
	}

	responseData := gin.H{
		"user":          userResp,
		"access_token":  token,
		"refresh_token": refreshToken,
	}

	response.Success(c, http.StatusOK, responseData, nil)
}

func (ctrl *Controller) Me(c *gin.Context) {
	// asumsi middleware sudah set userID di context
	log.Printf("auth context: %+v\n", c.Keys)

	userID, ok := c.Get("user_id")
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	userResp, err := ctrl.service.GetMe(
		c.Request.Context(),
		userID.(string),
	)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	response.Success(c, http.StatusOK, userResp, nil)
}

func (ctrl *Controller) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	// Response Success Seragam
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

func (ctrl *Controller) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := ctrl.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "REGISTER_FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

func (ctrl *Controller) RefreshToken(c *gin.Context) {
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
	newAccess, newRefresh, userResp, err := ctrl.service.RefreshToken(c.Request.Context(), refreshToken)
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
