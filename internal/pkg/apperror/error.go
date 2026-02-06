package apperror

import "fmt"

type AppError struct {
	Code       string // Error code (e.g., INVALID_INPUT)
	Message    string // User-friendly message
	HTTPStatus int    // HTTP status code
	Err        error  // Wrapped original error (optional)
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements errors.Unwrap interface for errors.Is/As
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError without wrapping
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        nil,
	}
}

// Wrap creates an AppError that wraps an existing error
func Wrap(err error, code, message string, httpStatus int) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}
