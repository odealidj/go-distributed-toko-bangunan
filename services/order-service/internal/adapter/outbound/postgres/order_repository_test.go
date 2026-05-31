package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

func TestOrderRepositoryIntegration_PersistsOrderAndOutboxInSameTransaction(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL tidak diset")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			inbox_events,
			outbox_events,
			saga_steps,
			saga_instances,
			order_items,
			orders
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	repository := NewOrderRepository(pool)
	order := model.Order{
		ID:              "ord_test_repo_1",
		CustomerName:    "Budi",
		CustomerPhone:   "081234567890",
		CustomerAddress: "Jl. Test 1",
		Status:          model.OrderStatusPending,
		TotalAmount:     136000,
		CorrelationID:   "corr_repo_1",
		Items: []model.OrderItem{
			{
				ID:          "ord_item_test_repo_1",
				OrderID:     "ord_test_repo_1",
				ProductID:   "prod_semen_50kg",
				ProductName: "Semen Portland 50kg",
				Unit:        "sak",
				Quantity:    2,
				UnitPrice:   68000,
				LineTotal:   136000,
			},
		},
	}
	sagaInstance := model.SagaInstance{
		ID:            "saga_test_repo_1",
		OrderID:       order.ID,
		Status:        model.SagaStatusStarted,
		CurrentStep:   model.SagaStepProductValidated,
		CorrelationID: order.CorrelationID,
	}
	step := model.SagaStep{
		ID:             "saga_step_test_repo_1",
		SagaID:         sagaInstance.ID,
		StepName:       model.SagaStepProductValidated,
		Status:         model.SagaStepStatusSucceeded,
		IdempotencyKey: "validate-products:ord_test_repo_1",
	}
	event := model.OutboxEvent{
		ID:            "evt_test_repo_1",
		AggregateID:   order.ID,
		AggregateType: "order",
		EventType:     "OrderCreated",
		CorrelationID: order.CorrelationID,
		CausationID:   "req_repo_1",
		Traceparent:   "00-11111111111111111111111111111111-2222222222222222-01",
		Payload:       []byte(`{"order_id":"ord_test_repo_1","status":"PENDING"}`),
		Status:        model.OutboxStatusPending,
	}

	if err := repository.CreateCheckout(ctx, order, sagaInstance, step, event); err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}

	storedOrder, err := repository.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if storedOrder.Status != model.OrderStatusPending {
		t.Fatalf("order status = %s, want %s", storedOrder.Status, model.OrderStatusPending)
	}
	if len(storedOrder.Items) != 1 {
		t.Fatalf("expected 1 order item, got %d", len(storedOrder.Items))
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'OrderCreated'`, order.ID).Scan(&outboxCount); err != nil {
		t.Fatalf("query outbox count: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("outbox count = %d, want 1", outboxCount)
	}

	if err := repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         order.ID,
		OrderStatus:     model.OrderStatusConfirmed,
		SagaID:          sagaInstance.ID,
		SagaStatus:      model.SagaStatusCompleted,
		SagaCurrentStep: model.SagaStepCommitStock,
		CompleteSaga:    true,
		Step: model.SagaStep{
			ID:             "saga_step_test_repo_2",
			SagaID:         sagaInstance.ID,
			StepName:       model.SagaStepCommitStock,
			Status:         model.SagaStepStatusSucceeded,
			IdempotencyKey: "commit-stock:ord_test_repo_1",
		},
		Event: model.OutboxEvent{
			ID:            "evt_test_repo_2",
			AggregateID:   order.ID,
			AggregateType: "order",
			EventType:     "OrderConfirmed",
			CorrelationID: order.CorrelationID,
			CausationID:   "evt_test_repo_1",
			Traceparent:   event.Traceparent,
			Payload:       []byte(`{"order_id":"ord_test_repo_1","status":"CONFIRMED"}`),
			Status:        model.OutboxStatusPending,
		},
	}); err != nil {
		t.Fatalf("RecordTransition: %v", err)
	}

	confirmedOrder, err := repository.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder after transition: %v", err)
	}
	if confirmedOrder.Status != model.OrderStatusConfirmed {
		t.Fatalf("order status after transition = %s, want %s", confirmedOrder.Status, model.OrderStatusConfirmed)
	}

	var confirmedCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'OrderConfirmed'`, order.ID).Scan(&confirmedCount); err != nil {
		t.Fatalf("query confirmed outbox count: %v", err)
	}
	if confirmedCount != 1 {
		t.Fatalf("confirmed outbox count = %d, want 1", confirmedCount)
	}
}

func TestOrderRepositoryIntegration_ProcessPaymentSucceededEventConfirmsOrderOnce(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL tidak diset")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			inbox_events,
			outbox_events,
			saga_steps,
			saga_instances,
			order_items,
			orders
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	repository := NewOrderRepository(pool)
	order := model.Order{
		ID:              "ord_test_repo_payment_1",
		CustomerName:    "Budi",
		CustomerPhone:   "081234567890",
		CustomerAddress: "Jl. Test 2",
		Status:          model.OrderStatusPending,
		TotalAmount:     136000,
		CorrelationID:   "corr_repo_payment_1",
		Items: []model.OrderItem{
			{
				ID:          "ord_item_test_repo_payment_1",
				OrderID:     "ord_test_repo_payment_1",
				ProductID:   "prod_semen_50kg",
				ProductName: "Semen Portland 50kg",
				Unit:        "sak",
				Quantity:    2,
				UnitPrice:   68000,
				LineTotal:   136000,
			},
		},
	}
	sagaInstance := model.SagaInstance{
		ID:            "saga_test_repo_payment_1",
		OrderID:       order.ID,
		Status:        model.SagaStatusStarted,
		CurrentStep:   model.SagaStepProductValidated,
		CorrelationID: order.CorrelationID,
	}
	step := model.SagaStep{
		ID:             "saga_step_test_repo_payment_1",
		SagaID:         sagaInstance.ID,
		StepName:       model.SagaStepProductValidated,
		Status:         model.SagaStepStatusSucceeded,
		IdempotencyKey: "validate-products:ord_test_repo_payment_1",
	}
	event := model.OutboxEvent{
		ID:            "evt_test_repo_payment_1",
		AggregateID:   order.ID,
		AggregateType: "order",
		EventType:     "OrderCreated",
		CorrelationID: order.CorrelationID,
		CausationID:   "req_repo_payment_1",
		Payload:       []byte(`{"order_id":"ord_test_repo_payment_1","status":"PENDING"}`),
		Status:        model.OutboxStatusPending,
	}
	if err := repository.CreateCheckout(ctx, order, sagaInstance, step, event); err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}

	if err := repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         order.ID,
		OrderStatus:     model.OrderStatusWaitingPayment,
		PaymentID:       "pay_test_repo_payment_1",
		SagaID:          sagaInstance.ID,
		SagaStatus:      model.SagaStatusPaymentCreated,
		SagaCurrentStep: model.SagaStepCreatePayment,
		Step: model.SagaStep{
			ID:             "saga_step_test_repo_payment_2",
			SagaID:         sagaInstance.ID,
			StepName:       model.SagaStepCreatePayment,
			Status:         model.SagaStepStatusSucceeded,
			IdempotencyKey: "create-payment:ord_test_repo_payment_1",
		},
	}); err != nil {
		t.Fatalf("RecordTransition waiting payment: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"order_id":   order.ID,
		"payment_id": "pay_test_repo_payment_1",
		"status":     "SUCCEEDED",
	})
	duplicate, err := repository.ProcessPaymentEvent(ctx, messaging.EventEnvelope{
		EventID:       "evt_payment_succeeded_1",
		EventType:     "PaymentSucceeded",
		AggregateID:   order.ID,
		AggregateType: "payment",
		CorrelationID: order.CorrelationID,
		Payload:       json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("ProcessPaymentEvent first call: %v", err)
	}
	if duplicate {
		t.Fatal("first payment event should not be duplicate")
	}

	duplicate, err = repository.ProcessPaymentEvent(ctx, messaging.EventEnvelope{
		EventID:       "evt_payment_succeeded_1",
		EventType:     "PaymentSucceeded",
		AggregateID:   order.ID,
		AggregateType: "payment",
		CorrelationID: order.CorrelationID,
		Payload:       json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("ProcessPaymentEvent duplicate call: %v", err)
	}
	if !duplicate {
		t.Fatal("duplicate payment event should be detected")
	}

	confirmedOrder, err := repository.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder after payment event: %v", err)
	}
	if confirmedOrder.Status != model.OrderStatusConfirmed {
		t.Fatalf("order status = %s, want %s", confirmedOrder.Status, model.OrderStatusConfirmed)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'OrderConfirmed'`, order.ID).Scan(&outboxCount); err != nil {
		t.Fatalf("query outbox confirmed count: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("outbox OrderConfirmed count = %d, want 1", outboxCount)
	}

	var inboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_events WHERE event_id = $1`, "evt_payment_succeeded_1").Scan(&inboxCount); err != nil {
		t.Fatalf("query inbox count: %v", err)
	}
	if inboxCount != 1 {
		t.Fatalf("inbox count = %d, want 1", inboxCount)
	}
}
