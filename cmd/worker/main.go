package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"

	"go-gadget-api/internal/dbgen"
	"go-gadget-api/internal/outbox"
)

func main() {
	log.Println("[WORKER] Starting outbox processor...")

	// 1. Connect to database
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
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

	// 3. Create outbox repository
	outboxRepo := outbox.NewRepository(db)

	// 4. Start processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go processOutboxEvents(ctx, outboxRepo, kafkaWriter)

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[WORKER] Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Println("[WORKER] Stopped")
}

func processOutboxEvents(ctx context.Context, repo outbox.Repository, writer *kafka.Writer) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("[WORKER] Outbox processor started (polling every 5s)")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processPendingEvents(ctx, repo, writer); err != nil {
				log.Printf("[WORKER] Error processing events: %v", err)
			}
		}
	}
}

func processPendingEvents(ctx context.Context, repo outbox.Repository, writer *kafka.Writer) error {
	// Get pending events
	events, err := repo.GetPendingEvents(ctx, 10)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	log.Printf("[WORKER] Processing %d pending events", len(events))

	for _, event := range events {
		if err := publishEvent(ctx, writer, event); err != nil {
			log.Printf("[WORKER] Failed to publish event %s: %v", event.ID, err)
			_ = repo.UpdateEventStatus(ctx, event.ID, "FAILED")
		} else {
			_ = repo.UpdateEventStatus(ctx, event.ID, "PROCESSED")
			log.Printf("[WORKER] Event %s published successfully", event.ID)
		}
	}

	return nil
}

func publishEvent(ctx context.Context, writer *kafka.Writer, event dbgen.OutboxEvent) error {
	msg := kafka.Message{
		Key:   []byte(event.AggregateID.String()),
		Value: event.Payload,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
		},
	}

	return writer.WriteMessages(ctx, msg)
}
