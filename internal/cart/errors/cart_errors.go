package carterrors

import (
	"net/http"

	"go-gadget-api/internal/pkg/apperror"
)

var (
	// ========================
	// Generic Cart Errors
	// ========================

	ErrInvalidCartInput = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid cart input",
		http.StatusBadRequest,
	)

	ErrCartNotFound = apperror.New(
		apperror.CodeNotFound,
		"Cart not found",
		http.StatusNotFound,
	)

	ErrCartItemNotFound = apperror.New(
		apperror.CodeNotFound,
		"Cart item not found",
		http.StatusNotFound,
	)

	// ========================
	// Quantity Errors
	// ========================

	ErrInvalidQty = apperror.New(
		apperror.CodeInvalidInput,
		"Invalid quantity",
		http.StatusBadRequest,
	)

	ErrQtyMustBeGreaterThanZero = apperror.New(
		apperror.CodeInvalidInput,
		"Quantity must be greater than zero",
		http.StatusBadRequest,
	)

	// ========================
	// Business Rule Errors
	// ========================

	ErrProductAlreadyInCart = apperror.New(
		apperror.CodeConflict,
		"Product already exists in cart",
		http.StatusConflict,
	)

	ErrCartIsEmpty = apperror.New(
		apperror.CodeInvalidInput,
		"Cart is empty",
		http.StatusBadRequest,
	)
)
