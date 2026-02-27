package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit // jumlah request per detik
	b   int        // burst (kapasitas kantong)
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(key string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[key]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[key] = limiter
	}

	return limiter
}

func RateLimitByIP(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(r, b) // Gunakan struct IPRateLimiter dari penjelasan sebelumnya
	return func(c *gin.Context) {
		if !limiter.GetLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests from this IP"})
			return
		}
		c.Next()
	}
}

// RateLimitByUser: r = request per detik, b = burst
func RateLimitByUser(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(r, b)
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			userID = c.GetString("user_id_validated")
		}
		if userID == "" {
			c.Next() // Jika belum login, skip ke middleware berikutnya (atau bisa ditolak)
			return
		}
		if !limiter.GetLimiter(userID).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests from this user"})
			return
		}
		c.Next()
	}
}
