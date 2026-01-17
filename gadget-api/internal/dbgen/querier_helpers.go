package dbgen

import (
	"database/sql"
	"strings"
)

// NewNullString mengonversi string menjadi sql.NullString yang valid
func NewNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{
			String: "",
			Valid:  false,
		}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

// NewNullInt32 mengonversi int32 menjadi sql.NullInt32
func NewNullInt32(i int32) sql.NullInt32 {
	return sql.NullInt32{
		Int32: i,
		Valid: true,
	}
}

func NewNullBool(v bool) sql.NullBool {
	return sql.NullBool{Bool: v, Valid: true}
}

func ToText(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{
			String: "",
			Valid:  false,
		}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
