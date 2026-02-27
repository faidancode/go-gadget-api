package autherrors

import (
	"net/http"

	"go-gadget-api/internal/pkg/apperror"
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

	ErrResetTokenInvalid = apperror.New(
		apperror.CodeUnauthorized,
		"Reset password link is invalid or has expired",
		http.StatusUnauthorized,
	)

	ErrResetTokenExpired = apperror.New(
		apperror.CodeUnauthorized,
		"Reset password link has expired. Please request a new one",
		http.StatusUnauthorized,
	)

	ErrConfirmationTokenInvalid = apperror.New(
		apperror.CodeUnauthorized,
		"Email confirmation link is invalid or has expired",
		http.StatusUnauthorized,
	)

	ErrConfirmationTokenExpired = apperror.New(
		apperror.CodeUnauthorized,
		"Email confirmation link has expired. Please request a new one",
		http.StatusUnauthorized,
	)

	ErrConfirmationPinInvalid = apperror.New(
		apperror.CodeUnauthorized,
		"Invalid confirmation PIN",
		http.StatusUnauthorized,
	)
)
