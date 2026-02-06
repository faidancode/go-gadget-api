package carterrors

import "github.com/go-playground/validator/v10"

func MapValidationError(err error) error {
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			switch e.Field() {

			case "ProductID":
				return ErrInvalidCartInput

			case "Qty":
				if e.Tag() == "min" {
					return ErrQtyMustBeGreaterThanZero
				}
				return ErrInvalidQty

			case "Price":
				return ErrInvalidCartInput
			}
		}
	}

	return ErrInvalidCartInput
}
