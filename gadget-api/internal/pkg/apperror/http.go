package apperror

import (
	"errors"
	"net/http"
)

type HTTPError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func ToHTTP(err error) *HTTPError {
	if err == nil {
		return &HTTPError{
			Status:  http.StatusOK,
			Code:    "",
			Message: "",
			Details: nil,
		}
	}

	var appErr *AppError
	// errors.As akan mencari AppError di dalam chain error
	if errors.As(err, &appErr) {
		return &HTTPError{
			Status:  appErr.HTTPStatus,
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: nil,
		}
	}

	return &HTTPError{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternalError,
		Message: "internal server error",
		Details: nil,
	}
}
