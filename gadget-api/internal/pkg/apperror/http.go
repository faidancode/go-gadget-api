package apperror

import "net/http"

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

	if appErr, ok := err.(*AppError); ok {
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
