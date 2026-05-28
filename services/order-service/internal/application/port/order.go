package port

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
)

type OrderRepository interface {
	Ping(ctx context.Context) error
	CreateCheckout(ctx context.Context, order model.Order, saga model.SagaInstance, step model.SagaStep, event model.OutboxEvent) error
	GetOrder(ctx context.Context, orderID string) (model.Order, error)
	RecordTransition(ctx context.Context, transition model.SagaTransition) error
}

type InventoryClient interface {
	ValidateProducts(ctx context.Context, items []model.OrderItemInput, correlationID, causationID string) ([]model.ValidatedOrderItem, int64, error)
	ReserveStock(ctx context.Context, orderID string, items []model.OrderItemInput, correlationID, causationID, idempotencyKey string) error
	ReleaseStock(ctx context.Context, orderID, correlationID, causationID, idempotencyKey string) error
	CommitStock(ctx context.Context, orderID, correlationID, causationID, idempotencyKey string) error
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, orderID string, amount int64, paymentMode, correlationID, causationID, idempotencyKey string) (model.Payment, error)
}
