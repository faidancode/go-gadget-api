package consumer

import (
	"context"
	"go-gadget-api/internal/cart"
	"log"

	"github.com/segmentio/kafka-go"
)

func ConsumeMessages(ctx context.Context, reader *kafka.Reader, cartService cart.Service) {
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
