// internal/brand/brand_errors.go
package branderrors

import (
	"gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidUUID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid brand ID",
		http.StatusBadRequest,
	)

	ErrBrandNotFound = apperror.New(
		apperror.CodeNotFound,
		"Brand not found",
		http.StatusNotFound,
	)

	ErrBrandFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process brand operation",
		http.StatusInternalServerError,
	)

	ErrImageUploadFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to upload brand image",
		http.StatusInternalServerError,
	)

	ErrImageDeleteFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to delete brand image",
		http.StatusInternalServerError,
	)

	ErrInvalidImageURL = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid brand image URL",
		http.StatusBadRequest,
	)
)
