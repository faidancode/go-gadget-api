package app

import (
	"context"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/messaging/kafka/consumer"

	"go-gadget-api/internal/shared/connection"
	"go-gadget-api/internal/shared/database/dbgen"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
)

func RunConsumer() error {
	log.Println("[CONSUMER] Starting cart consumer...")

	// 1. Connect to database
	db, err := connection.ConnectDBWithRetry(os.Getenv("DB_URL"), 5)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("[CONSUMER] Database connected")

	queries := dbgen.New(db)

	cartRepo := cart.NewRepository(queries)
	cartService := cart.NewService(db, cartRepo)

	// Setup Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_BROKER")},
		Topic:   "order.events",
		GroupID: "cart-consumer-group",
	})
	defer reader.Close()
	log.Println("[CONSUMER] Kafka reader initialized")

	// Start consuming
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.ConsumeMessages(ctx, reader, cartService)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[CONSUMER] Shutting down...")
	cancel()
	log.Println("[CONSUMER] Stopped")

	return nil
}
