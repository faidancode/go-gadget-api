package customer

import "errors"

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrEmailAlreadyUsed = errors.New("email already registered")
)
