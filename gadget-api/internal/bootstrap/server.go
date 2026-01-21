package bootstrap

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// StartHTTPServer menjalankan Gin server dengan graceful shutdown
func StartHTTPServer(
	router *gin.Engine,
	cfg ServerConfig,
	auditLogger AuditLogger,
) {
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Println("🚀 HTTP server running on port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Println("🛑 Shutdown signal received:", sig.String())

	// Audit log BEFORE shutdown
	auditLogger.Log(context.Background(), AuditLog{
		Action:  "SERVER_SHUTDOWN",
		Message: "Server is shutting down",
		Meta: map[string]any{
			"signal": sig.String(),
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("❌ Forced shutdown:", err)
	} else {
		log.Println("✅ Server exited gracefully")
	}
}
