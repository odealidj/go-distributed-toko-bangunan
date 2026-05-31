package port

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type OrderRepository interface {
	Ping(ctx context.Context) error
	CreateCheckout(ctx context.Context, order model.Order, saga model.SagaInstance, step model.SagaStep, event model.OutboxEvent) error
	GetOrder(ctx context.Context, orderID string) (model.Order, error)
	ListOrders(ctx context.Context, filter model.OrderFilter) ([]model.Order, int, error)
	CancelOrder(ctx context.Context, command model.CancelOrderCommand) (model.Order, error)
	ProcessPaymentEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error)
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

type OutboxRepository interface {
	ListPending(ctx context.Context, limit int32) ([]model.OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string) error
}
