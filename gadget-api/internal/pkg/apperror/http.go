package apperror

import (
	"errors"
	"net/http"
)

type HTTPError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ToHTTP converts any error to HTTPError
func ToHTTP(err error) *HTTPError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &HTTPError{
			Status:  appErr.HTTPStatus,
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: nil,
		}
	}

	// Fallback for unknown errors
	return &HTTPError{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternalError,
		Message: "An unexpected error occurred",
		Details: nil,
	}
}
