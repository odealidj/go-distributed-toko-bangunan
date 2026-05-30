package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

func NewPaymentEventsConsumer(cfg config.ServiceConfig, order *usecase.OrderUseCase) (*messaging.Consumer, error) {
	return messaging.NewConsumer(
		cfg.KafkaBrokers,
		messaging.TopicPaymentEvents,
		"order-service.payment-events-consumer",
		func(ctx context.Context, event messaging.EventEnvelope) error {
			duplicate, err := order.ProcessPaymentEvent(ctx, event)
			if err != nil {
				return err
			}
			if duplicate {
				slog.InfoContext(ctx, "duplicate kafka event skipped", "event_id", event.EventID, "event_type", event.EventType, "order_id", event.AggregateID)
				return nil
			}
			slog.InfoContext(ctx, "payment event processed", "event_id", event.EventID, "event_type", event.EventType, "order_id", event.AggregateID)
			return nil
		},
		messaging.ConsumerOptions{
			ServiceName:    cfg.ServiceName,
			MaxRetries:     cfg.KafkaMaxRetries,
			InitialBackoff: time.Duration(cfg.KafkaBackoffMs) * time.Millisecond,
			DLQSuffix:      cfg.KafkaDLQSuffix,
		},
	)
}
