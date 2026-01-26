package producterrors

import (
	"gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidProductID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid product ID",
		http.StatusBadRequest,
	)

	ErrInvalidCategoryID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid category ID",
		http.StatusBadRequest,
	)

	ErrProductNotFound = apperror.New(
		apperror.CodeNotFound,
		"Product not found",
		http.StatusNotFound,
	)

	ErrCategoryNotFound = apperror.New(
		apperror.CodeNotFound,
		"Category not found",
		http.StatusNotFound,
	)

	ErrProductFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process product operation",
		http.StatusInternalServerError,
	)

	ErrImageUploadFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to upload image",
		http.StatusInternalServerError,
	)
)
