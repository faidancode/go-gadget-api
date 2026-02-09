package app

import (
	"context"
	"go-gadget-api/internal/messaging/kafka/producer"
	"go-gadget-api/internal/outbox"
	"go-gadget-api/internal/shared/connection"
	"go-gadget-api/internal/shared/database/dbgen"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func RunWorker() error {
	log.Println("[WORKER] Starting outbox processor...")

	// 1. Connect to database
	db, err := connection.ConnectDBWithRetry(os.Getenv("DB_URL"), 5)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("[WORKER] Database connected")

	// 2. Setup Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_BROKER")),
		Topic:    "order.events",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()
	log.Println("[WORKER] Kafka writer initialized")
	queries := dbgen.New(db)

	// 3. Create outbox repository
	outboxRepo := outbox.NewRepository(queries)

	// 4. Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go producer.ProcessOutboxEvents(ctx, outboxRepo, kafkaWriter)

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[WORKER] Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Println("[WORKER] Stopped")

	return nil
}
