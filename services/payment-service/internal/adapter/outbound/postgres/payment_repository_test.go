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
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE payment_attempts, payments RESTART IDENTITY CASCADE`); err != nil {
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
	})
	if err != nil {
		t.Fatalf("create manual payment: %v", err)
	}
	if manual.Status != model.PaymentStatusPending {
		t.Fatalf("expected pending status, got %s", manual.Status)
	}

	cancelled, err := repository.CancelPayment(ctx, model.CancelPaymentCommand{
		PaymentID: manual.ID,
		Reason:    "demo_cancel",
	})
	if err != nil {
		t.Fatalf("cancel payment: %v", err)
	}
	if cancelled.Status != model.PaymentStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}

	cancelledAgain, err := repository.CancelPayment(ctx, model.CancelPaymentCommand{
		PaymentID: manual.ID,
		Reason:    "duplicate_cancel",
	})
	if err != nil {
		t.Fatalf("cancel payment duplicate: %v", err)
	}
	if cancelledAgain.Status != model.PaymentStatusCancelled {
		t.Fatalf("expected duplicate cancel stay cancelled, got %s", cancelledAgain.Status)
	}
}
