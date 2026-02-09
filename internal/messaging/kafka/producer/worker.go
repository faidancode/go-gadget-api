package producer

import (
	"context"
	"go-gadget-api/internal/outbox"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func ProcessOutboxEvents(ctx context.Context, repo outbox.Repository, writer *kafka.Writer) {
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
	events, err := repo.ListPending(ctx, 10)
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
			_ = repo.MarkFailed(ctx, event.ID)
			continue
		}

		if err := repo.MarkSent(ctx, event.ID); err != nil {
			log.Printf("[WORKER] Failed to mark event %s as SENT: %v", event.ID, err)
			continue
		}

		log.Printf("[WORKER] Event %s sent and marked successfully", event.ID)
	}

	return nil
}
