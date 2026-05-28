package port

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
)

type PaymentRepository interface {
	Ping(ctx context.Context) error
	CreatePayment(ctx context.Context, command model.CreatePaymentCommand) (model.Payment, error)
	GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error)
	CancelPayment(ctx context.Context, command model.CancelPaymentCommand) (model.Payment, error)
}
