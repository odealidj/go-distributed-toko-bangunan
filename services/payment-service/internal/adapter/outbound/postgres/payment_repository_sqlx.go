package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
)

type PaymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, command model.CreatePaymentCommand) (model.Payment, error) {
	existing, err := r.findByIdempotencyKey(ctx, command.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, model.ErrPaymentNotFound) {
		return model.Payment{}, err
	}

	existing, err = r.findByOrderID(ctx, command.OrderID)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, model.ErrPaymentNotFound) {
		return model.Payment{}, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
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
		return model.Payment{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.Payment{}, err
	}
	return payment, nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error) {
	var payment paymentRow
	if err := r.db.GetContext(ctx, &payment, `
		SELECT id, order_id, amount, status, payment_mode, idempotency_key
		FROM payments
		WHERE id = $1
	`, paymentID); err != nil {
		if isNotFound(err) {
			return model.Payment{}, model.ErrPaymentNotFound
		}
		return model.Payment{}, err
	}
	return payment.toModel(), nil
}

func (r *PaymentRepository) CancelPayment(ctx context.Context, command model.CancelPaymentCommand) (model.Payment, error) {
	payment, err := r.findForCancel(ctx, command)
	if err != nil {
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
		return model.Payment{}, err
	}
	defer rollback(tx)

	if _, err := tx.ExecContext(ctx, `
		UPDATE payments
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, payment.ID, model.PaymentStatusCancelled); err != nil {
		return model.Payment{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payment_attempts (id, payment_id, status, reason)
		VALUES ($1, $2, $3, $4)
	`, newID("pay_attempt"), payment.ID, model.PaymentStatusCancelled, nullableString(command.Reason)); err != nil {
		return model.Payment{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.Payment{}, err
	}

	payment.Status = model.PaymentStatusCancelled
	return payment, nil
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
