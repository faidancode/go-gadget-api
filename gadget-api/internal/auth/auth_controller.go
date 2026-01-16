package auth

import (
	"gadget-api/internal/pkg/response"
	"net/http"

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

	token, userResp, err := ctrl.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "Username atau password salah", nil)
		return
	}

	// Set Cookie
	c.SetCookie(
		"access_token",
		token,
		86400,
		"/",
		"",
		false,
		true,
	)

	// Response Success Seragam
	// Data yang dikirim adalah struct AuthResponse (Username & Role)
	response.Success(c, http.StatusOK, "Login berhasil", userResp)
}

func (ctrl *Controller) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	// Response Success Seragam
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}
