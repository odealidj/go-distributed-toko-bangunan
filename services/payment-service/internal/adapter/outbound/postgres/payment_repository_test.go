package postgres

import (
	"context"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
)

func TestPaymentRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL tidak diset")
	}

	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE inbox_events, outbox_events, payment_attempts, payments RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	repository := NewPaymentRepository(db)

	created, err := repository.CreatePayment(ctx, model.CreatePaymentCommand{
		OrderID:        "ord_test_1",
		Amount:         150000,
		PaymentMode:    model.PaymentModeFailure,
		IdempotencyKey: "idem_test_1",
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if created.Status != model.PaymentStatusFailed {
		t.Fatalf("expected status %s, got %s", model.PaymentStatusFailed, created.Status)
	}
	assertOutboxCount(t, db, "ord_test_1", "PaymentFailed", 1)

	duplicate, err := repository.CreatePayment(ctx, model.CreatePaymentCommand{
		OrderID:        "ord_test_1",
		Amount:         150000,
		PaymentMode:    model.PaymentModeFailure,
		IdempotencyKey: "idem_test_1",
	})
	if err != nil {
		t.Fatalf("duplicate create payment: %v", err)
	}
	if duplicate.ID != created.ID {
		t.Fatalf("expected same payment id, got %s vs %s", duplicate.ID, created.ID)
	}

	manual, err := repository.CreatePayment(ctx, model.CreatePaymentCommand{
		OrderID:        "ord_test_2",
		Amount:         200000,
		PaymentMode:    model.PaymentModeManual,
		IdempotencyKey: "idem_test_2",
		CorrelationID:  "corr_test_2",
		CausationID:    "req_test_2",
	})
	if err != nil {
		t.Fatalf("create manual payment: %v", err)
	}
	if manual.Status != model.PaymentStatusPending {
		t.Fatalf("expected pending status, got %s", manual.Status)
	}
	assertOutboxCount(t, db, "ord_test_2", "PaymentCreated", 1)

	succeeded, err := repository.SucceedPayment(ctx, model.CompletePaymentCommand{
		PaymentID:     manual.ID,
		Reason:        "manual_success",
		CorrelationID: "corr_test_2",
		CausationID:   "req_test_2",
	})
	if err != nil {
		t.Fatalf("succeed payment: %v", err)
	}
	if succeeded.Status != model.PaymentStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", succeeded.Status)
	}
	assertOutboxCount(t, db, "ord_test_2", "PaymentSucceeded", 1)

	manualCancel, err := repository.CreatePayment(ctx, model.CreatePaymentCommand{
		OrderID:        "ord_test_3",
		Amount:         210000,
		PaymentMode:    model.PaymentModeManual,
		IdempotencyKey: "idem_test_3",
		CorrelationID:  "corr_test_3",
		CausationID:    "req_test_3",
	})
	if err != nil {
		t.Fatalf("create cancelable manual payment: %v", err)
	}
	if manualCancel.Status != model.PaymentStatusPending {
		t.Fatalf("expected pending status, got %s", manualCancel.Status)
	}

	cancelled, err := repository.CancelPayment(ctx, model.CancelPaymentCommand{
		PaymentID:     manualCancel.ID,
		Reason:        "demo_cancel",
		CorrelationID: "corr_test_3",
		CausationID:   "req_test_3",
	})
	if err != nil {
		t.Fatalf("cancel payment: %v", err)
	}
	if cancelled.Status != model.PaymentStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}
	assertOutboxCount(t, db, "ord_test_3", "PaymentCancelled", 1)

	cancelledAgain, err := repository.CancelPayment(ctx, model.CancelPaymentCommand{
		PaymentID:     manualCancel.ID,
		Reason:        "duplicate_cancel",
		CorrelationID: "corr_test_3",
		CausationID:   "req_test_3",
	})
	if err != nil {
		t.Fatalf("cancel payment duplicate: %v", err)
	}
	if cancelledAgain.Status != model.PaymentStatusCancelled {
		t.Fatalf("expected duplicate cancel stay cancelled, got %s", cancelledAgain.Status)
	}
}

func assertOutboxCount(t *testing.T, db *sqlx.DB, orderID, eventType string, expected int) {
	t.Helper()

	var count int
	if err := db.GetContext(context.Background(), &count, `
		SELECT count(*)
		FROM outbox_events
		WHERE aggregate_id = $1 AND event_type = $2
	`, orderID, eventType); err != nil {
		t.Fatalf("query outbox %s: %v", eventType, err)
	}
	if count != expected {
		t.Fatalf("outbox count %s = %d, want %d", eventType, count, expected)
	}
}
