package auth

import (
	autherrors "go-gadget-api/internal/auth/errors"
	platform "go-gadget-api/internal/pkg/request"
	"go-gadget-api/internal/pkg/response"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	logger  *zap.Logger
}

func NewHandler(s *Service, logger ...*zap.Logger) *Handler {
	l := zap.L().Named("auth.handler")
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0].Named("auth.handler")
	}
	return &Handler{service: s, logger: l}
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
	h.logger.Debug("http register request started")

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("http register validation failed", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	// Menambahkan context email agar log berikutnya lebih informatif
	logger := h.logger.With(zap.String("email", req.Email))

	res, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		logger.Error("http register service failed", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "REGISTER_FAILED", err.Error(), nil)
		return
	}

	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	// Attempt to send verification email
	verification, verifyErr := h.service.RequestEmailConfirmation(c.Request.Context(), req.Email, clientType)
	if verifyErr != nil {
		// Log sebagai Error karena email verifikasi sangat penting untuk aktivasi akun
		logger.Error("http register verification email failed to send",
			zap.Error(verifyErr),
			zap.String("user_id", res.ID),
		)

		responseData := gin.H{
			"user":                    res,
			"verification_email_sent": false,
			"verification_error":      verifyErr.Error(),
		}
		response.Success(c, http.StatusCreated, responseData, nil)
		return
	}

	logger.Info("http register success",
		zap.String("user_id", res.ID),
		zap.Bool("email_sent", verification.EmailSent),
	)

	responseData := gin.H{
		"user":                    res,
		"verification_email_sent": verification.EmailSent,
	}
	response.Success(c, http.StatusCreated, responseData, nil)
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

func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := h.service.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "PASSWORD_RESET_REQUEST_FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		status := http.StatusBadRequest
		code := "RESET_PASSWORD_FAILED"
		if err == autherrors.ErrResetTokenInvalid || err == autherrors.ErrResetTokenExpired {
			status = http.StatusUnauthorized
			code = "RESET_PASSWORD_TOKEN_INVALID"
		}
		response.Error(c, status, code, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) RequestEmailConfirmation(c *gin.Context) {
	var req RequestEmailConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	res, err := h.service.RequestEmailConfirmation(c.Request.Context(), req.Email, clientType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "EMAIL_CONFIRMATION_REQUEST_FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) ResendEmailConfirmation(c *gin.Context) {
	var req RequestEmailConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	clientHeader := c.GetHeader("X-Client-Type")
	userAgent := c.GetHeader("User-Agent")
	clientType := platform.ResolveClientType(clientHeader, userAgent)

	res, err := h.service.ResendEmailConfirmation(c.Request.Context(), req.Email, clientType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "EMAIL_CONFIRMATION_RESEND_FAILED", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) ConfirmEmailByToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "token is required", nil)
		return
	}

	res, err := h.service.ConfirmEmailByToken(c.Request.Context(), token)
	if err != nil {
		status := http.StatusBadRequest
		code := "EMAIL_CONFIRMATION_FAILED"
		if err == autherrors.ErrConfirmationTokenInvalid || err == autherrors.ErrConfirmationTokenExpired {
			status = http.StatusUnauthorized
			code = "EMAIL_CONFIRMATION_TOKEN_INVALID"
		}
		response.Error(c, status, code, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) ConfirmEmailByPin(c *gin.Context) {
	var req ConfirmEmailByPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	res, err := h.service.ConfirmEmailByPin(c.Request.Context(), req.Email, req.PIN)
	if err != nil {
		status := http.StatusBadRequest
		code := "EMAIL_CONFIRMATION_PIN_FAILED"
		if err == autherrors.ErrConfirmationPinInvalid || err == autherrors.ErrConfirmationTokenExpired {
			status = http.StatusUnauthorized
			code = "EMAIL_CONFIRMATION_PIN_INVALID"
		}
		response.Error(c, status, code, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}
