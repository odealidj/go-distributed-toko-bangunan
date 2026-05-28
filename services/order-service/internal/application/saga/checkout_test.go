package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/saga"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
)

func TestCreateCheckoutSuccessConfirmsOrderAndCommitsStock(t *testing.T) {
	repository := newFakeRepository()
	inventory := &fakeInventory{}
	payment := &fakePayment{status: model.PaymentStatusSucceeded}
	orchestrator := saga.NewCheckoutOrchestrator(repository, inventory, payment)

	order, err := orchestrator.CreateCheckout(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}

	if order.Status != model.OrderStatusConfirmed {
		t.Fatalf("order status = %s, want %s", order.Status, model.OrderStatusConfirmed)
	}
	if !inventory.commitCalled {
		t.Fatal("expected inventory commit stock to be called")
	}
	if inventory.releaseCalled {
		t.Fatal("release stock should not be called on success")
	}
}

func TestCreateCheckoutPaymentFailedCancelsOrderAndReleasesStock(t *testing.T) {
	repository := newFakeRepository()
	inventory := &fakeInventory{}
	payment := &fakePayment{status: model.PaymentStatusFailed}
	orchestrator := saga.NewCheckoutOrchestrator(repository, inventory, payment)

	order, err := orchestrator.CreateCheckout(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}

	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("order status = %s, want %s", order.Status, model.OrderStatusCancelled)
	}
	if !inventory.releaseCalled {
		t.Fatal("expected inventory release stock to be called")
	}
	if inventory.commitCalled {
		t.Fatal("commit stock should not be called when payment failed")
	}
}

func TestCreateCheckoutReserveFailedRejectsOrder(t *testing.T) {
	repository := newFakeRepository()
	inventory := &fakeInventory{reserveErr: model.ErrInsufficientStock}
	payment := &fakePayment{status: model.PaymentStatusSucceeded}
	orchestrator := saga.NewCheckoutOrchestrator(repository, inventory, payment)

	order, err := orchestrator.CreateCheckout(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}

	if order.Status != model.OrderStatusRejected {
		t.Fatalf("order status = %s, want %s", order.Status, model.OrderStatusRejected)
	}
	if payment.called {
		t.Fatal("payment should not be called when stock reservation failed")
	}
}

func validCommand() model.CreateCheckoutCommand {
	return model.CreateCheckoutCommand{
		CustomerName:  "Budi",
		CustomerPhone: "08123456789",
		PaymentMode:   model.PaymentModeSuccess,
		CorrelationID: "corr_test",
		CausationID:   "req_test",
		Items: []model.OrderItemInput{
			{ProductID: "prod_semen_50kg", Quantity: 2},
		},
	}
}

type fakeRepository struct {
	order model.Order
	steps []model.SagaStep
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{}
}

func (r *fakeRepository) Ping(context.Context) error {
	return nil
}

func (r *fakeRepository) CreateCheckout(_ context.Context, order model.Order, _ model.SagaInstance, step model.SagaStep, _ model.OutboxEvent) error {
	r.order = order
	r.steps = append(r.steps, step)
	return nil
}

func (r *fakeRepository) GetOrder(_ context.Context, orderID string) (model.Order, error) {
	if r.order.ID != orderID {
		return model.Order{}, model.ErrOrderNotFound
	}
	return r.order, nil
}

func (r *fakeRepository) RecordTransition(_ context.Context, transition model.SagaTransition) error {
	if r.order.ID != transition.OrderID {
		return model.ErrOrderNotFound
	}
	if transition.OrderStatus != "" {
		r.order.Status = transition.OrderStatus
	}
	if transition.PaymentID != "" {
		r.order.PaymentID = transition.PaymentID
	}
	if transition.Step.ID != "" {
		r.steps = append(r.steps, transition.Step)
	}
	return nil
}

type fakeInventory struct {
	reserveErr    error
	releaseCalled bool
	commitCalled  bool
}

func (f *fakeInventory) ValidateProducts(context.Context, []model.OrderItemInput, string, string) ([]model.ValidatedOrderItem, int64, error) {
	return []model.ValidatedOrderItem{
		{
			ProductID:   "prod_semen_50kg",
			ProductName: "Semen 50kg",
			Unit:        "sak",
			Quantity:    2,
			UnitPrice:   68000,
			LineTotal:   136000,
		},
	}, 136000, nil
}

func (f *fakeInventory) ReserveStock(context.Context, string, []model.OrderItemInput, string, string, string) error {
	return f.reserveErr
}

func (f *fakeInventory) ReleaseStock(context.Context, string, string, string, string) error {
	f.releaseCalled = true
	return nil
}

func (f *fakeInventory) CommitStock(context.Context, string, string, string, string) error {
	f.commitCalled = true
	return nil
}

type fakePayment struct {
	status string
	called bool
	err    error
}

func (f *fakePayment) CreatePayment(context.Context, string, int64, string, string, string, string) (model.Payment, error) {
	f.called = true
	if f.err != nil {
		return model.Payment{}, f.err
	}
	if f.status == "" {
		return model.Payment{}, errors.New("payment status not configured")
	}
	return model.Payment{ID: "pay_test", Status: f.status}, nil
}
