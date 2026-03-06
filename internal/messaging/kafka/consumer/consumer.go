package consumer

import (
	"context"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/email"
	"go-gadget-api/internal/shared/database/dbgen"
	"log"

	"github.com/segmentio/kafka-go"
)

func ConsumeMessages(ctx context.Context, reader *kafka.Reader, cartService cart.Service, emailSvc email.Service, queries *dbgen.Queries) {
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
		aggregateType := getHeader(msg.Headers, "aggregate_type")

		log.Printf("[CONSUMER] Received message. Key: %s, EventType: %s, AggregateType: %s", string(msg.Key), eventType, aggregateType)

		for _, h := range msg.Headers {
			log.Printf("[CONSUMER]   Header -> Key: %s, Value: %s", h.Key, string(h.Value))
		}

		if eventType == "DELETE_CART" {
			if err := handleDeleteCart(ctx, msg.Value, cartService); err != nil {
				log.Printf("[CONSUMER] Error handling DELETE_CART: %v", err)
			} else {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("[CONSUMER] Error committing message: %v", err)
				}
			}
		} else if eventType == "ORDER_STATUS_CHANGED" {
			if err := handleOrderStatusChanged(ctx, msg.Value, emailSvc, queries); err != nil {
				log.Printf("[CONSUMER] Error handling ORDER_STATUS_CHANGED: %v", err)
			} else {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("[CONSUMER] Error committing message: %v", err)
				}
			}
		} else if eventType == "ORDER_PAYMENT_UPDATED" {
			if err := handleOrderPaymentUpdated(ctx, msg.Value, emailSvc, queries); err != nil {
				log.Printf("[CONSUMER] Error handling ORDER_PAYMENT_UPDATED: %v", err)
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
