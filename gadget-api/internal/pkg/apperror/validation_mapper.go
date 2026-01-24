package apperror

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

func MapValidationError(err error) error {
	if errs, ok := err.(validator.ValidationErrors); ok {
		e := errs[0]
		switch e.Tag() {
		case "required":
			return RequiredField(strings.ToLower(e.Field()))
		default:
			return InvalidField(strings.ToLower(e.Field()))
		}
	}
	return New(
		CodeInvalidInput,
		"Invalid input",
		http.StatusBadRequest,
	)
}
