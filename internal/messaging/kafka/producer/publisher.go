package producer

import (
	"context"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/segmentio/kafka-go"
)

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
