package apperror

type AppError struct {
	Code       string // kode error unik (misal: INVALID_INPUT)
	Message    string // pesan yang user-friendly
	HTTPStatus int    // status code HTTP (misal: 400, 401)
	Err        error  // optional: wrapped error asli
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// New buat error baru tanpa wrapped error
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap buat membungkus error yang sudah ada
func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}
