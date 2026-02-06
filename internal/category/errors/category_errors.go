// internal/category/category_errors.go
package categoryerrors

import (
	"go-gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidUUID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid category ID",
		http.StatusBadRequest,
	)

	ErrCategoryNotFound = apperror.New(
		apperror.CodeNotFound,
		"category not found",
		http.StatusNotFound,
	)

	ErrCategoryFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process category operation",
		http.StatusInternalServerError,
	)

	ErrImageUploadFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to upload category image",
		http.StatusInternalServerError,
	)

	ErrImageDeleteFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to delete category image",
		http.StatusInternalServerError,
	)

	ErrInvalidImageURL = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid category image URL",
		http.StatusBadRequest,
	)
)
