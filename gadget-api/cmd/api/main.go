package main

import (
	"log"
	"os"
	"time"

	"gadget-api/internal/app"
	"gadget-api/internal/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	r := gin.Default()

	// build dependency + routes
	if err := app.BuildApp(r); err != nil {
		log.Fatal(err)
	}

	auditLogger := bootstrap.NewStdoutAuditLogger()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	bootstrap.StartHTTPServer(
		r,
		bootstrap.ServerConfig{
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		auditLogger,
	)
}
