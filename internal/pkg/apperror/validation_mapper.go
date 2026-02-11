package apperror

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// internal/pkg/apperror/validation_mapper.go
func formatFieldName(s string) string {
	// 1. Ganti underscore dengan spasi (recipient_phone -> recipient phone)
	s = strings.ReplaceAll(s, "_", " ")

	// 2. Ubah jadi Title Case (recipient phone -> Recipient Phone)
	caser := cases.Title(language.English)
	return caser.String(s)
}

func MapValidationError(err error) error {
	if errs, ok := err.(validator.ValidationErrors); ok {
		// Ambil error pertama
		e := errs[0]

		// e.Field() sekarang sudah otomatis 'recipient_phone'
		// karena kita sudah set RegisterTagNameFunc di apperror.Init()
		fieldName := e.Field()
		humanReadableField := formatFieldName(fieldName)

		switch e.Tag() {
		case "required":
			// Memanggil fungsi RequiredField yang mengembalikan *AppError
			// Pesannya akan menjadi: "recipient_phone is required"
			return RequiredField(humanReadableField)
		default:
			// Pesannya akan menjadi: "recipient_phone is invalid"
			return InvalidField(humanReadableField)
		}
	}

	return New(
		CodeInvalidInput,
		"Invalid input",
		http.StatusBadRequest,
	)
}
