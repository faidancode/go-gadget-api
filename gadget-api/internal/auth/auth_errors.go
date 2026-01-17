package auth

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
)
