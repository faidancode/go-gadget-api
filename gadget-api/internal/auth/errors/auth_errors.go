package autherrors

import (
	"net/http"

	"gadget-api/internal/pkg/apperror"
)

var (
	ErrUnauthorized = apperror.New(
		apperror.CodeUnauthorized,
		"Unauthorized access",
		http.StatusUnauthorized,
	)

	ErrInvalidToken = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid authentication token",
		http.StatusBadRequest,
	)

	ErrTokenExpired = apperror.New(
		apperror.CodeUnauthorized,
		"Authentication token expired",
		http.StatusUnauthorized,
	)

	ErrForbidden = apperror.New(
		apperror.CodeForbidden,
		"Access forbidden",
		http.StatusForbidden,
	)

	ErrUserNotFound = apperror.New(
		apperror.CodeNotFound,
		"User not found",
		http.StatusNotFound,
	)

	ErrRefreshTokenRequired = apperror.New(
		apperror.CodeUnauthorized,
		"Refresh token is required",
		http.StatusUnauthorized,
	)

	ErrInvalidRefreshToken = apperror.New(
		apperror.CodeUnauthorized,
		"Invalid or expired refresh token",
		http.StatusUnauthorized,
	)

	ErrSessionExpired = apperror.New(
		apperror.CodeUnauthorized,
		"Your session has expired, please login again",
		http.StatusUnauthorized,
	)

	ErrUnsupportedClient = apperror.New(
		apperror.CodeInvalidInput,
		"Unsupported client platform",
		http.StatusBadRequest,
	)

	// 🔥 Tambahan Auth-specific errors
	ErrInvalidCredentials = apperror.New(
		apperror.CodeUnauthorized,
		"Invalid email or password",
		http.StatusUnauthorized,
	)

	ErrEmailAlreadyRegistered = apperror.New(
		apperror.CodeConflict,
		"Email already registered",
		http.StatusConflict,
	)

	ErrTokenGenerationFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to generate authentication token",
		http.StatusInternalServerError,
	)

	ErrInvalidUserID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid user id",
		http.StatusBadRequest,
	)
)
