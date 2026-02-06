package wishlist

import (
	"go-gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidProductID = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid product ID",
		http.StatusBadRequest,
	)

	ErrProductNotFound = apperror.New(
		apperror.CodeNotFound,
		"Product not found",
		http.StatusNotFound,
	)

	ErrWishlistNotFound = apperror.New(
		apperror.CodeNotFound,
		"Wishlist not found",
		http.StatusNotFound,
	)

	ErrItemAlreadyExists = apperror.New(
		apperror.CodeConflict,
		"Item already in wishlist",
		http.StatusConflict,
	)

	ErrItemNotFound = apperror.New(
		apperror.CodeNotFound,
		"Item not found in wishlist",
		http.StatusNotFound,
	)

	ErrWishlistFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process wishlist operation",
		http.StatusInternalServerError,
	)
)
