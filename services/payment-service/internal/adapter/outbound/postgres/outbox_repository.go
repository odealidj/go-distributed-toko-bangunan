package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type PaymentOutboxRepository struct {
	db *sqlx.DB
}

var paymentOutboxTracer = otel.Tracer("payment-service/postgres/outbox")

func NewPaymentOutboxRepository(db *sqlx.DB) *PaymentOutboxRepository {
	return &PaymentOutboxRepository{db: db}
}

func (r *PaymentOutboxRepository) ListPending(ctx context.Context, limit int32) ([]model.OutboxEvent, error) {
	ctx, span := paymentOutboxTracer.Start(ctx, "postgres.PaymentOutboxRepository.ListPending")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.Int("outbox.limit", int(limit)),
	)

	type outboxRow struct {
		ID            string    `db:"id"`
		AggregateID   string    `db:"aggregate_id"`
		AggregateType string    `db:"aggregate_type"`
		EventType     string    `db:"event_type"`
		CorrelationID string    `db:"correlation_id"`
		CausationID   *string   `db:"causation_id"`
		Traceparent   *string   `db:"traceparent"`
		Payload       []byte    `db:"payload"`
		Status        string    `db:"status"`
		CreatedAt     time.Time `db:"created_at"`
	}

	rows := make([]outboxRow, 0)
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT id, aggregate_id, aggregate_type, event_type, correlation_id, causation_id, traceparent, payload, status, created_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC, id ASC
		LIMIT $1
	`, limit); err != nil {
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
			CausationID:   deref(row.CausationID),
			Traceparent:   deref(row.Traceparent),
			Payload:       row.Payload,
			Status:        row.Status,
		})
	}
	return events, nil
}

func (r *PaymentOutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	ctx, span := paymentOutboxTracer.Start(ctx, "postgres.PaymentOutboxRepository.MarkPublished")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "UPDATE"),
		attribute.String("event_id", eventID),
	)
	if _, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = 'PUBLISHED', published_at = now()
		WHERE id = $1
	`, eventID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *PaymentOutboxRepository) MarkFailed(ctx context.Context, eventID string) error {
	ctx, span := paymentOutboxTracer.Start(ctx, "postgres.PaymentOutboxRepository.MarkFailed")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "UPDATE"),
		attribute.String("event_id", eventID),
	)
	if _, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET retry_count = retry_count + 1
		WHERE id = $1
	`, eventID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
