package saga

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type CheckoutOrchestrator struct {
	repository port.OrderRepository
	inventory  port.InventoryClient
	payment    port.PaymentClient
}

var sagaTracer = otel.Tracer("order-service/saga")

func NewCheckoutOrchestrator(repository port.OrderRepository, inventory port.InventoryClient, payment port.PaymentClient) *CheckoutOrchestrator {
	return &CheckoutOrchestrator{
		repository: repository,
		inventory:  inventory,
		payment:    payment,
	}
}

func (o *CheckoutOrchestrator) Ping(ctx context.Context) error {
	return o.repository.Ping(ctx)
}

func (o *CheckoutOrchestrator) CreateCheckout(ctx context.Context, command model.CreateCheckoutCommand) (model.Order, error) {
	command = normalizeCommand(command)
	ctx = observability.WithCorrelationID(ctx, command.CorrelationID)
	ctx, span := sagaTracer.Start(ctx, "CheckoutSaga.CreateCheckout")
	defer span.End()
	span.SetAttributes(attribute.String("correlation_id", command.CorrelationID))

	if err := validateCommand(command); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	validatedItems, totalAmount, err := o.inventory.ValidateProducts(ctx, command.Items, command.CorrelationID, command.CausationID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	orderID := newID("ord")
	sagaID := newID("saga")
	order := model.Order{
		ID:              orderID,
		CustomerName:    command.CustomerName,
		CustomerPhone:   command.CustomerPhone,
		CustomerAddress: command.CustomerAddress,
		Status:          model.OrderStatusPending,
		TotalAmount:     totalAmount,
		CorrelationID:   command.CorrelationID,
		Items:           orderItems(orderID, validatedItems),
	}
	span.SetAttributes(attribute.String("order_id", orderID))
	sagaInstance := model.SagaInstance{
		ID:            sagaID,
		OrderID:       orderID,
		Status:        model.SagaStatusStarted,
		CurrentStep:   model.SagaStepProductValidated,
		CorrelationID: command.CorrelationID,
	}
	if err := o.repository.CreateCheckout(
		ctx,
		order,
		sagaInstance,
		sagaStep(sagaID, model.SagaStepProductValidated, model.SagaStepStatusSucceeded, "validate-products:"+orderID, ""),
		outboxEvent(ctx, orderID, "OrderCreated", command.CorrelationID, command.CausationID, orderCreatedPayload(order)),
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	reserveKey := "reserve-stock:" + orderID + ":checkout"
	if err := o.inventory.ReserveStock(ctx, orderID, command.Items, command.CorrelationID, command.CausationID, reserveKey); err != nil {
		if errors.Is(err, model.ErrInsufficientStock) {
			return o.rejectOrder(ctx, orderID, sagaID, command, err.Error())
		}
		return o.markManualRepair(ctx, orderID, sagaID, command, model.SagaStepReserveStock, err)
	}
	if err := o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		OrderStatus:     model.OrderStatusStockReserved,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusStockReserved,
		SagaCurrentStep: model.SagaStepReserveStock,
		Step:            sagaStep(sagaID, model.SagaStepReserveStock, model.SagaStepStatusSucceeded, reserveKey, ""),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	paymentKey := "create-payment:" + orderID + ":checkout"
	payment, err := o.payment.CreatePayment(ctx, orderID, totalAmount, command.PaymentMode, command.CorrelationID, command.CausationID, paymentKey)
	if err != nil {
		return o.compensatePaymentFailure(ctx, orderID, sagaID, command, "", err.Error())
	}
	if err := o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		OrderStatus:     model.OrderStatusWaitingPayment,
		PaymentID:       payment.ID,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusPaymentCreated,
		SagaCurrentStep: model.SagaStepCreatePayment,
		Step:            sagaStep(sagaID, model.SagaStepCreatePayment, model.SagaStepStatusSucceeded, paymentKey, ""),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	switch payment.Status {
	case model.PaymentStatusSucceeded:
		return o.confirmOrder(ctx, orderID, sagaID, command)
	case model.PaymentStatusFailed:
		return o.compensatePaymentFailure(ctx, orderID, sagaID, command, payment.ID, "payment failed")
	default:
		return o.repository.GetOrder(ctx, orderID)
	}
}

func (o *CheckoutOrchestrator) GetOrder(ctx context.Context, orderID string) (model.Order, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return model.Order{}, model.ErrInvalidInput
	}
	return o.repository.GetOrder(ctx, orderID)
}

func (o *CheckoutOrchestrator) rejectOrder(ctx context.Context, orderID, sagaID string, command model.CreateCheckoutCommand, reason string) (model.Order, error) {
	ctx, span := sagaTracer.Start(ctx, "CheckoutSaga.RejectOrder")
	defer span.End()
	span.SetAttributes(attribute.String("order_id", orderID), attribute.String("correlation_id", command.CorrelationID))

	err := o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		OrderStatus:     model.OrderStatusRejected,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusCompensated,
		SagaCurrentStep: model.SagaStepReserveStock,
		CompleteSaga:    true,
		Step:            sagaStep(sagaID, model.SagaStepReserveStock, model.SagaStepStatusFailed, "reserve-stock:"+orderID+":checkout", reason),
		Event:           outboxEvent(ctx, orderID, "OrderRejected", command.CorrelationID, command.CausationID, orderStatusPayload(orderID, model.OrderStatusRejected, reason)),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	return o.repository.GetOrder(ctx, orderID)
}

func (o *CheckoutOrchestrator) confirmOrder(ctx context.Context, orderID, sagaID string, command model.CreateCheckoutCommand) (model.Order, error) {
	ctx, span := sagaTracer.Start(ctx, "CheckoutSaga.ConfirmOrder")
	defer span.End()
	span.SetAttributes(attribute.String("order_id", orderID), attribute.String("correlation_id", command.CorrelationID))

	commitKey := "commit-stock:" + orderID + ":payment-succeeded"
	if err := o.inventory.CommitStock(ctx, orderID, command.CorrelationID, command.CausationID, commitKey); err != nil {
		return o.markManualRepair(ctx, orderID, sagaID, command, model.SagaStepCommitStock, err)
	}
	err := o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		OrderStatus:     model.OrderStatusConfirmed,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusCompleted,
		SagaCurrentStep: model.SagaStepCommitStock,
		CompleteSaga:    true,
		Step:            sagaStep(sagaID, model.SagaStepCommitStock, model.SagaStepStatusSucceeded, commitKey, ""),
		Event:           outboxEvent(ctx, orderID, "OrderConfirmed", command.CorrelationID, command.CausationID, orderStatusPayload(orderID, model.OrderStatusConfirmed, "")),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	return o.repository.GetOrder(ctx, orderID)
}

func (o *CheckoutOrchestrator) compensatePaymentFailure(ctx context.Context, orderID, sagaID string, command model.CreateCheckoutCommand, paymentID, reason string) (model.Order, error) {
	ctx, span := sagaTracer.Start(ctx, "CheckoutSaga.CompensatePaymentFailure")
	defer span.End()
	span.SetAttributes(attribute.String("order_id", orderID), attribute.String("correlation_id", command.CorrelationID))

	releaseKey := "release-stock:" + orderID + ":payment-failed"
	if err := o.inventory.ReleaseStock(ctx, orderID, command.CorrelationID, command.CausationID, releaseKey); err != nil {
		return o.markManualRepair(ctx, orderID, sagaID, command, model.SagaStepReleaseStock, err)
	}
	err := o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		OrderStatus:     model.OrderStatusCancelled,
		PaymentID:       paymentID,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusCompensated,
		SagaCurrentStep: model.SagaStepReleaseStock,
		CompleteSaga:    true,
		Step:            sagaStep(sagaID, model.SagaStepReleaseStock, model.SagaStepStatusSucceeded, releaseKey, ""),
		Event:           outboxEvent(ctx, orderID, "OrderCancelled", command.CorrelationID, command.CausationID, orderStatusPayload(orderID, model.OrderStatusCancelled, reason)),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	return o.repository.GetOrder(ctx, orderID)
}

func (o *CheckoutOrchestrator) markManualRepair(ctx context.Context, orderID, sagaID string, command model.CreateCheckoutCommand, stepName string, cause error) (model.Order, error) {
	_, span := sagaTracer.Start(ctx, "CheckoutSaga.MarkManualRepair")
	defer span.End()
	span.SetAttributes(attribute.String("order_id", orderID), attribute.String("correlation_id", command.CorrelationID), attribute.String("saga_step", stepName))
	span.RecordError(cause)
	span.SetStatus(codes.Error, cause.Error())

	_ = o.repository.RecordTransition(ctx, model.SagaTransition{
		OrderID:         orderID,
		SagaID:          sagaID,
		SagaStatus:      model.SagaStatusFailedManualRepair,
		SagaCurrentStep: stepName,
		Step:            sagaStep(sagaID, stepName, model.SagaStepStatusFailed, stepName+":"+orderID+":checkout", cause.Error()),
	})
	return model.Order{}, cause
}

func normalizeCommand(command model.CreateCheckoutCommand) model.CreateCheckoutCommand {
	command.CustomerName = strings.TrimSpace(command.CustomerName)
	command.CustomerPhone = strings.TrimSpace(command.CustomerPhone)
	command.CustomerAddress = strings.TrimSpace(command.CustomerAddress)
	command.PaymentMode = strings.ToUpper(strings.TrimSpace(command.PaymentMode))
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	if command.PaymentMode == "" {
		command.PaymentMode = model.PaymentModeSuccess
	}
	if command.CorrelationID == "" {
		command.CorrelationID = newID("corr")
	}
	if command.CausationID == "" {
		command.CausationID = command.CorrelationID
	}
	return command
}

func validateCommand(command model.CreateCheckoutCommand) error {
	if command.CustomerName == "" || command.CustomerPhone == "" || len(command.Items) == 0 {
		return model.ErrInvalidInput
	}
	if command.PaymentMode != model.PaymentModeSuccess && command.PaymentMode != model.PaymentModeFailure && command.PaymentMode != model.PaymentModeManual {
		return model.ErrInvalidInput
	}
	for _, item := range command.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 {
			return model.ErrInvalidInput
		}
	}
	return nil
}

func orderItems(orderID string, items []model.ValidatedOrderItem) []model.OrderItem {
	result := make([]model.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, model.OrderItem{
			ID:          newID("ord_item"),
			OrderID:     orderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}
	return result
}

func sagaStep(sagaID, stepName, status, idempotencyKey, errorMessage string) model.SagaStep {
	return model.SagaStep{
		ID:             newID("saga_step"),
		SagaID:         sagaID,
		StepName:       stepName,
		Status:         status,
		IdempotencyKey: idempotencyKey,
		ErrorMessage:   errorMessage,
	}
}

func outboxEvent(ctx context.Context, orderID, eventType, correlationID, causationID string, payload any) model.OutboxEvent {
	raw, _ := json.Marshal(payload)
	return model.OutboxEvent{
		ID:            newID("evt"),
		AggregateID:   orderID,
		AggregateType: "order",
		EventType:     eventType,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Traceparent:   observability.TraceparentFromContext(ctx),
		Payload:       raw,
		Status:        model.OutboxStatusPending,
	}
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_local"
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

type orderCreatedEventPayload struct {
	OrderID     string                  `json:"order_id"`
	Status      string                  `json:"status"`
	TotalAmount int64                   `json:"total_amount"`
	Items       []orderItemEventPayload `json:"items"`
}

type orderItemEventPayload struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	LineTotal   int64   `json:"line_total"`
}

type orderStatusEventPayload struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

func orderCreatedPayload(order model.Order) orderCreatedEventPayload {
	items := make([]orderItemEventPayload, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, orderItemEventPayload{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}
	return orderCreatedEventPayload{
		OrderID:     order.ID,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		Items:       items,
	}
}

func orderStatusPayload(orderID, status, reason string) orderStatusEventPayload {
	return orderStatusEventPayload{
		OrderID: orderID,
		Status:  status,
		Reason:  reason,
	}
}
