package kafka

import (
	"context"
	"log/slog"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

func NewOrderEventsConsumer(cfg config.ServiceConfig, payment *usecase.PaymentUseCase) (*messaging.Consumer, error) {
	return messaging.NewConsumer(
		cfg.KafkaBrokers,
		messaging.TopicOrderEvents,
		"payment-service.order-events-consumer",
		func(ctx context.Context, event messaging.EventEnvelope) error {
			duplicate, err := payment.ProcessOrderEvent(ctx, event)
			if err != nil {
				return err
			}
			if duplicate {
				slog.InfoContext(ctx, "duplicate kafka event skipped", "event_id", event.EventID, "event_type", event.EventType, "order_id", event.AggregateID)
				return nil
			}
			slog.InfoContext(ctx, "kafka event processed", "event_id", event.EventID, "event_type", event.EventType, "order_id", event.AggregateID)
			return nil
		},
	)
}
