package review

import (
	"gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidReviewID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid review ID",
		http.StatusBadRequest,
	)

	ErrInvalidProductSlug = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid product slug",
		http.StatusBadRequest,
	)

	ErrReviewNotFound = apperror.New(
		apperror.CodeNotFound,
		"Review not found",
		http.StatusNotFound,
	)

	ErrProductNotFound = apperror.New(
		apperror.CodeNotFound,
		"Product not found",
		http.StatusNotFound,
	)

	ErrReviewAlreadyExists = apperror.New(
		apperror.CodeConflict,
		"You have already reviewed this product",
		http.StatusConflict,
	)

	ErrNotPurchased = apperror.New(
		apperror.CodeInvalidState,
		"You must purchase this product before reviewing",
		http.StatusForbidden,
	)

	ErrOrderNotCompleted = apperror.New(
		apperror.CodeInvalidState,
		"Your order must be completed before you can review",
		http.StatusForbidden,
	)

	ErrUnauthorizedReview = apperror.New(
		apperror.CodeUnauthorized,
		"You are not authorized to modify this review",
		http.StatusForbidden,
	)

	ErrReviewFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process review operation",
		http.StatusInternalServerError,
	)
)
