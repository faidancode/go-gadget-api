package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"

	"go-gadget-api/internal/cart"
)

type DeleteCartPayload struct {
	UserID string `json:"user_id"`
}

func main() {
	log.Println("[CONSUMER] Starting cart consumer...")

	// 1. Connect to database
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("[CONSUMER] Database connected")

	// 2. Setup cart service
	cartRepo := cart.NewRepository(db)
	cartService := cart.NewService(db, cartRepo)

	// 3. Setup Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_BROKER")},
		Topic:   "order.events",
		GroupID: "cart-consumer-group",
	})
	defer reader.Close()
	log.Println("[CONSUMER] Kafka reader initialized")

	// 4. Start consuming
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumeMessages(ctx, reader, cartService)

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[CONSUMER] Shutting down...")
	cancel()
	log.Println("[CONSUMER] Stopped")
}

func consumeMessages(ctx context.Context, reader *kafka.Reader, cartService cart.Service) {
	log.Println("[CONSUMER] Started consuming messages")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[CONSUMER] Error fetching message: %v", err)
			continue
		}

		eventType := getHeader(msg.Headers, "event_type")

		if eventType == "DELETE_CART" {
			if err := handleDeleteCart(ctx, msg.Value, cartService); err != nil {
				log.Printf("[CONSUMER] Error handling DELETE_CART: %v", err)
			} else {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("[CONSUMER] Error committing message: %v", err)
				}
			}
		} else {
			// Skip unknown event types
			_ = reader.CommitMessages(ctx, msg)
		}
	}
}

func handleDeleteCart(ctx context.Context, payload []byte, cartService cart.Service) error {
	var data DeleteCartPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Deleting cart for user: %s", data.UserID)

	if err := cartService.ClearCart(ctx, data.UserID); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Cart deleted successfully for user: %s", data.UserID)
	return nil
}

func getHeader(headers []kafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}
