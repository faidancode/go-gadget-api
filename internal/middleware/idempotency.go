package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// middleware/idempotency.go

func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		idempKey := c.GetHeader("Idempotency-Key")
		userID := c.GetString("user_id_validated")
		fmt.Printf("[IDEMPOTENCY MIDDLEWARE] idempKey: '%s', userID: '%s'\n", idempKey, userID) // ← Debug

		if idempKey == "" || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		cacheKey := fmt.Sprintf("idemp:%s:%s:%s", c.FullPath(), userID, idempKey)
		lockKey := cacheKey + ":lock" // Key khusus untuk locking

		// 1. CEK CACHE (Seperti sebelumnya)
		val, err := rdb.Get(c.Request.Context(), cacheKey).Result()
		if err == nil {
			var cachedRes any
			json.Unmarshal([]byte(val), &cachedRes)
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"status": "success", "data": cachedRes})
			return
		}

		// 2. ATOMIC LOCK (SetNX)
		// Mencoba membuat key 'lock'. Jika sudah ada, berarti request lain sedang jalan.
		// Set expiry pendek (misal 30 detik) agar jika server crash, lock otomatis hilang.
		isNew, _ := rdb.SetNX(c.Request.Context(), lockKey, "locked", 30*time.Second).Result()

		if !isNew {
			// Request ganda terdeteksi saat proses masih berlangsung!
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"code":    "PROCESSING",
				"message": "Transaksi Anda sedang diproses, mohon tunggu sebentar.",
			})
			return
		}

		// Tambahkan lockKey ke context agar bisa dihapus oleh Handler setelah selesai
		c.Set("idempotency_cache_key", cacheKey)
		c.Set("idempotency_lock_key", lockKey)

		c.Next()
	}
}
