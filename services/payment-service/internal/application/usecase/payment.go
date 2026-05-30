package usecase

import (
	"context"
	"strings"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type PaymentUseCase struct {
	repository port.PaymentRepository
}

func NewPayment(repository port.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repository: repository}
}

func (u *PaymentUseCase) Ping(ctx context.Context) error {
	return u.repository.Ping(ctx)
}

func (u *PaymentUseCase) CreatePayment(ctx context.Context, command model.CreatePaymentCommand) (model.Payment, error) {
	command.OrderID = strings.TrimSpace(command.OrderID)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.PaymentMode = normalizePaymentMode(command.PaymentMode)
	if command.OrderID == "" || command.IdempotencyKey == "" || command.Amount <= 0 || command.PaymentMode == "" {
		return model.Payment{}, model.ErrInvalidInput
	}
	return u.repository.CreatePayment(ctx, command)
}

func (u *PaymentUseCase) GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error) {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return model.Payment{}, model.ErrInvalidInput
	}
	return u.repository.GetPaymentByID(ctx, paymentID)
}

func (u *PaymentUseCase) CancelPayment(ctx context.Context, command model.CancelPaymentCommand) (model.Payment, error) {
	command.PaymentID = strings.TrimSpace(command.PaymentID)
	command.OrderID = strings.TrimSpace(command.OrderID)
	command.Reason = strings.TrimSpace(command.Reason)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	if command.PaymentID == "" && command.OrderID == "" {
		return model.Payment{}, model.ErrInvalidInput
	}
	return u.repository.CancelPayment(ctx, command)
}

func (u *PaymentUseCase) SucceedPayment(ctx context.Context, command model.CompletePaymentCommand) (model.Payment, error) {
	command.PaymentID = strings.TrimSpace(command.PaymentID)
	command.Reason = strings.TrimSpace(command.Reason)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	if command.PaymentID == "" {
		return model.Payment{}, model.ErrInvalidInput
	}
	return u.repository.SucceedPayment(ctx, command)
}

func (u *PaymentUseCase) FailPayment(ctx context.Context, command model.CompletePaymentCommand) (model.Payment, error) {
	command.PaymentID = strings.TrimSpace(command.PaymentID)
	command.Reason = strings.TrimSpace(command.Reason)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	if command.PaymentID == "" {
		return model.Payment{}, model.ErrInvalidInput
	}
	return u.repository.FailPayment(ctx, command)
}

func (u *PaymentUseCase) ProcessOrderEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	return u.repository.ProcessOrderEvent(ctx, event)
}

func normalizePaymentMode(mode string) string {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case model.PaymentModeSuccess, model.PaymentModeFailure, model.PaymentModeManual:
		return mode
	default:
		return ""
	}
}
