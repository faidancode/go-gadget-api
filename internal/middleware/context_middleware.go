package middleware

import (
	"go-gadget-api/internal/shared/contextutil" // Sesuaikan dengan path project Anda

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ContextLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Handle Request ID
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Header("X-Request-ID", rid)

		// 2. Handle User ID (diambil dari middleware Auth sebelumnya)
		uid := c.GetString("user_id")

		// 3. Buat Scoped Logger yang sudah ditempeli Metadata
		// Logger ini yang akan digunakan di sepanjang request ini
		reqLogger := logger.With(
			zap.String("request_id", rid),
			zap.String("user_id", uid),
		)

		// 4. Propagasi ke Standard Context
		// Agar layer Service/Repo bisa ambil via contextutil tanpa tahu Gin
		ctx := c.Request.Context()
		ctx = contextutil.WithRequestID(ctx, rid)
		ctx = contextutil.WithUserID(ctx, uid)
		ctx = contextutil.WithLogger(ctx, reqLogger)

		// Update request dengan context baru
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
