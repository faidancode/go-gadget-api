package apperror

import "net/http"

func InvalidField(field string) *AppError {
	return New(
		CodeInvalidInput,
		field+" is invalid",
		http.StatusBadRequest,
	)
}

func RequiredField(field string) *AppError {
	return New(
		CodeInvalidInput,
		field+" is required",
		http.StatusBadRequest,
	)
}
