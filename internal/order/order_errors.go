package order

import (
	"go-gadget-api/internal/pkg/apperror"
	"net/http"
)

var (
	ErrInvalidOrderID = apperror.New(
		apperror.CodeInvalidInput,
		"invalid order id format",
		http.StatusBadRequest,
	)

	ErrInvalidStatusTransition = apperror.New(
		apperror.CodeInvalidState,
		"invalid status transition",
		http.StatusBadRequest,
	)

	ErrOrderNotFound = apperror.New(
		apperror.CodeNotFound,
		"order not found",
		http.StatusNotFound,
	)

	ErrCartEmpty = apperror.New(
		apperror.CodeInvalidState,
		"Your shopping cart is empty",
		http.StatusBadRequest,
	)

	ErrCannotCancel = apperror.New(
		apperror.CodeInvalidState,
		"Order cannot be cancelled",
		http.StatusBadRequest,
	)

	ErrOrderFailed = apperror.New(
		apperror.CodeInternalError,
		"Failed to process order, please try again",
		http.StatusInternalServerError,
	)

	ErrReceiptRequired = apperror.New(
		apperror.CodeInvalidInput,
		"receipt number is required for shipping",
		http.StatusBadRequest,
	)
)
