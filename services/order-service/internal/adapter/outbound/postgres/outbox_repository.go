package postgres

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
)

type OutboxRepository struct {
	queries *sqlc.Queries
}

func NewOutboxRepository(pool sqlc.DBTX) *OutboxRepository {
	return &OutboxRepository{queries: sqlc.New(pool)}
}

func (r *OutboxRepository) ListPending(ctx context.Context, limit int32) ([]model.OutboxEvent, error) {
	rows, err := r.queries.ListPendingOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	events := make([]model.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, model.OutboxEvent{
			ID:            row.ID,
			AggregateID:   row.AggregateID,
			AggregateType: row.AggregateType,
			EventType:     row.EventType,
			CorrelationID: row.CorrelationID,
			CausationID:   row.CausationID.String,
			Traceparent:   row.Traceparent.String,
			Payload:       row.Payload,
			Status:        row.Status,
			CreatedAt:     row.CreatedAt.Time,
		})
	}
	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	return r.queries.MarkOutboxEventPublished(ctx, eventID)
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID string) error {
	return r.queries.MarkOutboxEventFailed(ctx, eventID)
}
