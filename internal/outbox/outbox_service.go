package outbox

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

//go:generate mockgen -source=outbox_service.go -destination=../mock/outbox/outbox_service_mock.go -package=mock

type Service interface {
	Start(ctx context.Context)
}

type Processor struct {
	repo   Repository
	writer *kafka.Writer
}

func NewProcessor(repo Repository, writer *kafka.Writer) *Processor {
	return &Processor{
		repo:   repo,
		writer: writer,
	}
}

func (p *Processor) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			events, err := p.repo.ListPending(ctx, 10)
			if err != nil {
				log.Println("outbox fetch error:", err)
				continue
			}

			for _, e := range events {
				err := p.writer.WriteMessages(ctx, kafka.Message{
					Key:   []byte(e.EventType),
					Value: e.Payload,
				})
				if err != nil {
					log.Println("kafka publish failed:", err)
					continue
				}

				_ = p.repo.MarkSent(ctx, e.ID)
			}

		case <-ctx.Done():
			return
		}
	}
}
