package order

import (
	"gadget-api/internal/pkg/apperror"
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
		"keranjang kosong",
		http.StatusBadRequest,
	)

	ErrCannotCancel = apperror.New(
		apperror.CodeInvalidState,
		"order cannot be cancelled",
		http.StatusBadRequest,
	)
)
