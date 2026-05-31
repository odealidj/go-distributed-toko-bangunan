package usecase

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/saga"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type OrderUseCase struct {
	checkout *saga.CheckoutOrchestrator
}

func NewOrder(checkout *saga.CheckoutOrchestrator) *OrderUseCase {
	return &OrderUseCase{checkout: checkout}
}

func (u *OrderUseCase) Ping(ctx context.Context) error {
	return u.checkout.Ping(ctx)
}

func (u *OrderUseCase) CreateCheckout(ctx context.Context, command model.CreateCheckoutCommand) (model.Order, error) {
	return u.checkout.CreateCheckout(ctx, command)
}

func (u *OrderUseCase) GetOrder(ctx context.Context, orderID string) (model.Order, error) {
	return u.checkout.GetOrder(ctx, orderID)
}

func (u *OrderUseCase) ListOrders(ctx context.Context, filter model.OrderFilter) ([]model.Order, int, error) {
	return u.checkout.ListOrders(ctx, filter)
}

func (u *OrderUseCase) CancelOrder(ctx context.Context, command model.CancelOrderCommand) (model.Order, error) {
	return u.checkout.CancelOrder(ctx, command)
}

func (u *OrderUseCase) ProcessPaymentEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	return u.checkout.ProcessPaymentEvent(ctx, event)
}
