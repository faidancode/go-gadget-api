package helper

import (
	"database/sql"
	"strconv"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =======================
// RAW VALUE TO NULL (POSTGRES)
// =======================

// RawBoolToNull menerima bool biasa (true/false)
func RawBoolToNull(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

// RawStringToNull menerima string biasa
func RawStringToNull(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// RawInt32ToNull menerima int32 biasa
func RawInt32ToNull(i int32) sql.NullInt32 {
	return sql.NullInt32{Int32: i, Valid: true}
}

// =======================
// STRING
// =======================

func StringValue(s string) string {
	return s
}

func StringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func StringPtr(s string) *string {
	return &s
}

func StringToNull(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// =======================
// UUID (Postgres Native)
// =======================

// Mengonversi string ke google uuid
func StringToUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// Mengonversi pointer string ke google uuid (untuk optional field)
func StringPtrToUUID(s *string) uuid.UUID {
	if s == nil || *s == "" {
		return uuid.Nil
	}
	return StringToUUID(*s)
}

// =======================
// BOOL
// =======================

func BoolValue(b bool) bool {
	return b
}

func BoolPtrValue(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}

func BoolToNull(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// =======================
// INT32 / INT64
// =======================

func Int32Value(i int32) int32 {
	return i
}

func Int32PtrValue(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func Int32ToNull(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

// Tambahan: Postgres sering menggunakan BIGINT (int64)
func Int64ToNull(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// =======================
// DECIMAL (Postgres Numeric)
// =======================

func DecimalValue(d decimal.Decimal) decimal.Decimal {
	return d
}

func DecimalPtrValue(d *decimal.Decimal) decimal.Decimal {
	if d == nil {
		return decimal.Zero
	}
	return *d
}

// Untuk Postgres, sangat disarankan menggunakan String conversion
// untuk menjaga presisi tipe NUMERIC
func Float64ToDecimalExact(f float64) decimal.Decimal {
	return decimal.RequireFromString(
		strconv.FormatFloat(f, 'f', -1, 64),
	)
}

func Float64PtrToDecimalExact(f *float64) decimal.Decimal {
	if f == nil {
		return decimal.Zero
	}
	return Float64ToDecimalExact(*f)
}

// =======================
// DECIMAL → NULL
// =======================

func Float64ToNullDecimal(f *float64) decimal.NullDecimal {
	if f == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{
		Decimal: Float64ToDecimalExact(*f),
		Valid:   true,
	}
}

func DecimalToNull(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{
		Decimal: *d,
		Valid:   true,
	}
}
