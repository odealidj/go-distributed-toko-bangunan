package usecase

import (
	"context"
	"testing"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
)

func TestCreatePaymentNormalizesMode(t *testing.T) {
	repository := &fakeRepository{}
	uc := NewPayment(repository)

	_, err := uc.CreatePayment(context.Background(), model.CreatePaymentCommand{
		OrderID:        "ord_1",
		Amount:         100000,
		PaymentMode:    "success",
		IdempotencyKey: "idem_1",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repository.created.PaymentMode != model.PaymentModeSuccess {
		t.Fatalf("expected normalized mode %s, got %s", model.PaymentModeSuccess, repository.created.PaymentMode)
	}
}

func TestCancelPaymentRequiresIdentifier(t *testing.T) {
	repository := &fakeRepository{}
	uc := NewPayment(repository)

	_, err := uc.CancelPayment(context.Background(), model.CancelPaymentCommand{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type fakeRepository struct {
	created model.CreatePaymentCommand
}

func (r *fakeRepository) Ping(context.Context) error { return nil }

func (r *fakeRepository) CreatePayment(_ context.Context, command model.CreatePaymentCommand) (model.Payment, error) {
	r.created = command
	return model.Payment{
		ID:             "pay_1",
		OrderID:        command.OrderID,
		Amount:         command.Amount,
		Status:         model.PaymentStatusSucceeded,
		PaymentMode:    command.PaymentMode,
		IdempotencyKey: command.IdempotencyKey,
	}, nil
}

func (r *fakeRepository) GetPaymentByID(context.Context, string) (model.Payment, error) {
	return model.Payment{}, nil
}

func (r *fakeRepository) CancelPayment(context.Context, model.CancelPaymentCommand) (model.Payment, error) {
	return model.Payment{}, nil
}
