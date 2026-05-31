package postgres

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type OutboxRepository struct {
	queries *sqlc.Queries
}

var outboxRepositoryTracer = otel.Tracer("order-service/postgres/outbox")

func NewOutboxRepository(pool sqlc.DBTX) *OutboxRepository {
	return &OutboxRepository{queries: sqlc.New(pool)}
}

func (r *OutboxRepository) ListPending(ctx context.Context, limit int32) ([]model.OutboxEvent, error) {
	ctx, span := outboxRepositoryTracer.Start(ctx, "postgres.OutboxRepository.ListPending")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.Int("outbox.limit", int(limit)),
	)
	rows, err := r.queries.ListPendingOutboxEvents(ctx, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	ctx, span := outboxRepositoryTracer.Start(ctx, "postgres.OutboxRepository.MarkPublished")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "UPDATE"),
		attribute.String("event_id", eventID),
	)
	if err := r.queries.MarkOutboxEventPublished(ctx, eventID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID string) error {
	ctx, span := outboxRepositoryTracer.Start(ctx, "postgres.OutboxRepository.MarkFailed")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "UPDATE"),
		attribute.String("event_id", eventID),
	)
	if err := r.queries.MarkOutboxEventFailed(ctx, eventID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
