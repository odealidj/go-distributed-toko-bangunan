package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type PaymentRepository struct {
	db *sqlx.DB
}

var paymentRepositoryTracer = otel.Tracer("payment-service/postgres")

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Ping(ctx context.Context) error {
	ctx, span := paymentRepositoryTracer.Start(ctx, "postgres.PaymentRepository.Ping")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	if err := r.db.PingContext(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, command model.CreatePaymentCommand) (model.Payment, error) {
	ctx, span := paymentRepositoryTracer.Start(ctx, "postgres.PaymentRepository.CreatePayment")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("order_id", command.OrderID),
		attribute.String("idempotency_key", command.IdempotencyKey),
	)
	existing, err := r.findByIdempotencyKey(ctx, command.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, model.ErrPaymentNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	existing, err = r.findByOrderID(ctx, command.OrderID)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, model.ErrPaymentNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}
	defer rollback(tx)

	payment := model.Payment{
		ID:             newID("pay"),
		OrderID:        command.OrderID,
		Amount:         command.Amount,
		Status:         paymentStatusForMode(command.PaymentMode),
		PaymentMode:    command.PaymentMode,
		IdempotencyKey: command.IdempotencyKey,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payments (id, order_id, amount, status, payment_mode, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, payment.ID, payment.OrderID, payment.Amount, payment.Status, payment.PaymentMode, payment.IdempotencyKey); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	attempt := model.PaymentAttempt{
		ID:        newID("pay_attempt"),
		PaymentID: payment.ID,
		Status:    payment.Status,
		Reason:    attemptReasonForMode(command.PaymentMode),
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payment_attempts (id, payment_id, status, reason)
		VALUES ($1, $2, $3, $4)
	`, attempt.ID, attempt.PaymentID, attempt.Status, nullableString(attempt.Reason)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}
	return payment, nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error) {
	ctx, span := paymentRepositoryTracer.Start(ctx, "postgres.PaymentRepository.GetPaymentByID")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.String("payment_id", paymentID),
	)
	var payment paymentRow
	if err := r.db.GetContext(ctx, &payment, `
		SELECT id, order_id, amount, status, payment_mode, idempotency_key
		FROM payments
		WHERE id = $1
	`, paymentID); err != nil {
		if isNotFound(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return model.Payment{}, model.ErrPaymentNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}
	return payment.toModel(), nil
}

func (r *PaymentRepository) CancelPayment(ctx context.Context, command model.CancelPaymentCommand) (model.Payment, error) {
	ctx, span := paymentRepositoryTracer.Start(ctx, "postgres.PaymentRepository.CancelPayment")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "postgresql"), attribute.String("db.operation.name", "transaction"))
	payment, err := r.findForCancel(ctx, command)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}
	if payment.Status == model.PaymentStatusCancelled || payment.Status == model.PaymentStatusFailed {
		return payment, nil
	}
	if payment.Status == model.PaymentStatusSucceeded {
		return payment, nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}
	defer rollback(tx)

	if _, err := tx.ExecContext(ctx, `
		UPDATE payments
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, payment.ID, model.PaymentStatusCancelled); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payment_attempts (id, payment_id, status, reason)
		VALUES ($1, $2, $3, $4)
	`, newID("pay_attempt"), payment.ID, model.PaymentStatusCancelled, nullableString(command.Reason)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Payment{}, err
	}

	payment.Status = model.PaymentStatusCancelled
	return payment, nil
}

func (r *PaymentRepository) ProcessOrderEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	ctx, span := paymentRepositoryTracer.Start(ctx, "postgres.PaymentRepository.ProcessOrderEvent")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("event_id", event.EventID),
		attribute.String("event_type", event.EventType),
		attribute.String("order_id", event.AggregateID),
	)
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer rollback(tx)

	result, err := tx.ExecContext(ctx, `
		INSERT INTO inbox_events (event_id, event_type, aggregate_id, correlation_id, traceparent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING
	`, event.EventID, event.EventType, event.AggregateID, event.CorrelationID, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	if rowsAffected == 0 {
		if err := tx.Commit(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		return true, nil
	}

	if event.EventType == "OrderCancelled" {
		if err := cancelPaymentByOrderIDTx(ctx, tx, event.AggregateID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	return false, nil
}

func (r *PaymentRepository) findByIdempotencyKey(ctx context.Context, idempotencyKey string) (model.Payment, error) {
	var payment paymentRow
	if err := r.db.GetContext(ctx, &payment, `
		SELECT id, order_id, amount, status, payment_mode, idempotency_key
		FROM payments
		WHERE idempotency_key = $1
	`, idempotencyKey); err != nil {
		if isNotFound(err) {
			return model.Payment{}, model.ErrPaymentNotFound
		}
		return model.Payment{}, err
	}
	return payment.toModel(), nil
}

func (r *PaymentRepository) findByOrderID(ctx context.Context, orderID string) (model.Payment, error) {
	var payment paymentRow
	if err := r.db.GetContext(ctx, &payment, `
		SELECT id, order_id, amount, status, payment_mode, idempotency_key
		FROM payments
		WHERE order_id = $1
	`, orderID); err != nil {
		if isNotFound(err) {
			return model.Payment{}, model.ErrPaymentNotFound
		}
		return model.Payment{}, err
	}
	return payment.toModel(), nil
}

func (r *PaymentRepository) findForCancel(ctx context.Context, command model.CancelPaymentCommand) (model.Payment, error) {
	switch {
	case command.PaymentID != "":
		return r.GetPaymentByID(ctx, command.PaymentID)
	case command.OrderID != "":
		return r.findByOrderID(ctx, command.OrderID)
	default:
		return model.Payment{}, model.ErrInvalidInput
	}
}

type paymentRow struct {
	ID             string `db:"id"`
	OrderID        string `db:"order_id"`
	Amount         int64  `db:"amount"`
	Status         string `db:"status"`
	PaymentMode    string `db:"payment_mode"`
	IdempotencyKey string `db:"idempotency_key"`
}

func (r paymentRow) toModel() model.Payment {
	return model.Payment{
		ID:             r.ID,
		OrderID:        r.OrderID,
		Amount:         r.Amount,
		Status:         r.Status,
		PaymentMode:    r.PaymentMode,
		IdempotencyKey: r.IdempotencyKey,
	}
}

func paymentStatusForMode(mode string) string {
	switch mode {
	case model.PaymentModeSuccess:
		return model.PaymentStatusSucceeded
	case model.PaymentModeFailure:
		return model.PaymentStatusFailed
	default:
		return model.PaymentStatusPending
	}
}

func attemptReasonForMode(mode string) string {
	switch mode {
	case model.PaymentModeFailure:
		return "forced_failure"
	case model.PaymentModeSuccess:
		return "forced_success"
	default:
		return "waiting_manual_confirmation"
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func cancelPaymentByOrderIDTx(ctx context.Context, tx *sqlx.Tx, orderID string) error {
	var payment paymentRow
	if err := tx.GetContext(ctx, &payment, `
		SELECT id, order_id, amount, status, payment_mode, idempotency_key
		FROM payments
		WHERE order_id = $1
	`, orderID); err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	if payment.Status != model.PaymentStatusPending {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE payments
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, payment.ID, model.PaymentStatusCancelled); err != nil {
		return err
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO payment_attempts (id, payment_id, status, reason)
		VALUES ($1, $2, $3, $4)
	`, newID("pay_attempt"), payment.ID, model.PaymentStatusCancelled, nullableString("order_cancelled_event"))
	return err
}

func rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_local"
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
