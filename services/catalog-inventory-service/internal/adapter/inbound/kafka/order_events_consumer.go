package kafka

import (
	"context"
	"log/slog"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

func NewOrderEventsConsumer(cfg config.ServiceConfig, catalog *usecase.CatalogUseCase) (*messaging.Consumer, error) {
	return messaging.NewConsumer(
		cfg.KafkaBrokers,
		messaging.TopicOrderEvents,
		"catalog-inventory-service.order-events-consumer",
		func(ctx context.Context, event messaging.EventEnvelope) error {
			duplicate, err := catalog.ProcessOrderEvent(ctx, event)
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
