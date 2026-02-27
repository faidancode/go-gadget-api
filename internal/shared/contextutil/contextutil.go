package contextutil

import (
	"context"

	"go.uber.org/zap"
)

// contextKey adalah tipe privat agar tidak terjadi tabrakan key dengan library lain
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
	loggerKey    contextKey = "logger"
)

// --- Request ID Helpers ---

// WithRequestID memasukkan Request ID ke dalam context
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestIDKey, rid)
}

// GetRequestID mengambil Request ID dari context
func GetRequestID(ctx context.Context) string {
	if rid, ok := ctx.Value(requestIDKey).(string); ok {
		return rid
	}
	return ""
}

// --- User ID Helpers ---

// WithUserID memasukkan User ID ke dalam context
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, userIDKey, uid)
}

// GetUserID mengambil User ID dari context
func GetUserID(ctx context.Context) string {
	if uid, ok := ctx.Value(userIDKey).(string); ok {
		return uid
	}
	return ""
}

// --- Logger Helpers ---

// WithLogger memasukkan zap logger (yang biasanya sudah di-decorate) ke context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger mengambil logger dari context.
// Jika tidak ada, mengembalikan fallback (defaultLogger) agar tidak panic.
func GetLogger(ctx context.Context, defaultLogger *zap.Logger) *zap.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
			return l
		}
	}

	if defaultLogger != nil {
		return defaultLogger
	}

	// safety fallback agar tidak pernah nil
	return zap.NewNop()
}

// --- Combined Metadata (Optional but Useful) ---

// Metadata menampung info tracing dasar
type Metadata struct {
	RequestID string
	UserID    string
}

// ExtractMetadata mengambil semua info tracing sekaligus untuk kemudahan logging manual
func ExtractMetadata(ctx context.Context) Metadata {
	return Metadata{
		RequestID: GetRequestID(ctx),
		UserID:    GetUserID(ctx),
	}
}
