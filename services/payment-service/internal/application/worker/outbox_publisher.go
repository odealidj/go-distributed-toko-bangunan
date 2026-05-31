package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type OutboxPublisher struct {
	repository port.OutboxRepository
	producer   messaging.Producer
	interval   time.Duration
	batchSize  int32
}

func NewOutboxPublisher(repository port.OutboxRepository, producer messaging.Producer) *OutboxPublisher {
	return &OutboxPublisher{
		repository: repository,
		producer:   producer,
		interval:   time.Second,
		batchSize:  25,
	}
}

func (p *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		p.publishPending(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *OutboxPublisher) publishPending(ctx context.Context) {
	events, err := p.repository.ListPending(ctx, p.batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "gagal mengambil outbox pending", "error", err)
		return
	}

	for _, event := range events {
		envelope := messaging.EventEnvelope{
			EventID:       event.ID,
			EventType:     event.EventType,
			AggregateID:   event.AggregateID,
			AggregateType: event.AggregateType,
			CorrelationID: event.CorrelationID,
			CausationID:   event.CausationID,
			Traceparent:   event.Traceparent,
			Payload:       json.RawMessage(event.Payload),
		}
		topic := messaging.TopicForAggregate(event.AggregateType)
		if err := p.producer.Publish(ctx, topic, event.AggregateID, envelope); err != nil {
			_ = p.repository.MarkFailed(ctx, event.ID)
			slog.ErrorContext(ctx, "gagal publish outbox event", "event_id", event.ID, "event_type", event.EventType, "topic", topic, "order_id", event.AggregateID, "error", err)
			continue
		}
		if err := p.repository.MarkPublished(ctx, event.ID); err != nil {
			slog.ErrorContext(ctx, "gagal menandai outbox published", "event_id", event.ID, "order_id", event.AggregateID, "error", err)
			continue
		}
		slog.InfoContext(ctx, "outbox event published", "event_id", event.ID, "event_type", event.EventType, "topic", topic, "order_id", event.AggregateID)
	}
}
