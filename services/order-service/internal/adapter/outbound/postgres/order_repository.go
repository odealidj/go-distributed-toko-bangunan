package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type OrderRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var orderRepositoryTracer = otel.Tracer("order-service/postgres")

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *OrderRepository) Ping(ctx context.Context) error {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.Ping")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	if err := r.pool.Ping(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *OrderRepository) CreateCheckout(ctx context.Context, order model.Order, saga model.SagaInstance, step model.SagaStep, event model.OutboxEvent) error {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.CreateCheckout")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("order_id", order.ID),
		attribute.String("correlation_id", order.CorrelationID),
	)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer rollback(ctx, tx)

	q := r.queries.WithTx(tx)
	if err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
		ID:              order.ID,
		CustomerName:    order.CustomerName,
		CustomerPhone:   order.CustomerPhone,
		CustomerAddress: nullableText(order.CustomerAddress),
		Status:          order.Status,
		TotalAmount:     order.TotalAmount,
		PaymentID:       nullableText(order.PaymentID),
		CorrelationID:   order.CorrelationID,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, item := range order.Items {
		if err := q.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			ID:          item.ID,
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    numeric(item.Quantity),
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if err := q.CreateSagaInstance(ctx, sqlc.CreateSagaInstanceParams{
		ID:            saga.ID,
		OrderID:       saga.OrderID,
		Status:        saga.Status,
		CurrentStep:   saga.CurrentStep,
		CorrelationID: saga.CorrelationID,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := createSagaStep(ctx, q, step); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := createOutboxEvent(ctx, q, event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (model.Order, error) {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.GetOrder")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.String("order_id", orderID),
	)
	row, err := r.queries.GetOrder(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, model.ErrOrderNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	items, err := r.queries.ListOrderItems(ctx, orderID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	return orderFromRow(row, items), nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, filter model.OrderFilter) ([]model.Order, int, error) {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.ListOrders")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.String("order.status", filter.Status),
		attribute.Int("page", filter.Page),
		attribute.Int("per_page", filter.PerPage),
	)

	rows, err := r.queries.ListOrders(ctx, sqlc.ListOrdersParams{
		Column1: filter.Status,
		Limit:   int32(filter.PerPage),
		Offset:  int32((filter.Page - 1) * filter.PerPage),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}
	total, err := r.queries.CountOrders(ctx, filter.Status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}

	orders := make([]model.Order, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, orderFromRow(row, nil))
	}
	return orders, int(total), nil
}

func (r *OrderRepository) CancelOrder(ctx context.Context, command model.CancelOrderCommand) (model.Order, error) {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.CancelOrder")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("order_id", command.OrderID),
		attribute.String("correlation_id", command.CorrelationID),
	)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	defer rollback(ctx, tx)

	q := r.queries.WithTx(tx)
	orderRow, err := q.GetOrder(ctx, command.OrderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, model.ErrOrderNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	if !canCancelOrder(orderRow.Status) {
		return model.Order{}, model.ErrOrderConflict
	}

	sagaRow, err := q.GetSagaInstanceByOrderID(ctx, command.OrderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, model.ErrOrderNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	if err := q.UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
		ID:     command.OrderID,
		Status: model.OrderStatusCancelled,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	if err := q.UpdateSagaInstance(ctx, sqlc.UpdateSagaInstanceParams{
		OrderID:     command.OrderID,
		Status:      model.SagaStatusCompensated,
		CurrentStep: model.SagaStepCancelOrder,
		Column4:     true,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	if err := createSagaStep(ctx, q, model.SagaStep{
		ID:             newID("saga_step"),
		SagaID:         sagaRow.ID,
		StepName:       model.SagaStepCancelOrder,
		Status:         model.SagaStepStatusSucceeded,
		IdempotencyKey: "cancel-order:" + command.OrderID,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	if err := createOutboxEvent(ctx, q, orderCancelledEvent(command)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	items, err := q.ListOrderItems(ctx, command.OrderID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, err
	}

	orderRow.Status = model.OrderStatusCancelled
	return orderFromRow(orderRow, items), nil
}

func (r *OrderRepository) ProcessPaymentEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.ProcessPaymentEvent")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("event_id", event.EventID),
		attribute.String("event_type", event.EventType),
		attribute.String("order_id", event.AggregateID),
	)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer rollback(ctx, tx)

	result, err := tx.Exec(ctx, `
		INSERT INTO inbox_events (event_id, event_type, aggregate_id, correlation_id, traceparent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING
	`, event.EventID, event.EventType, event.AggregateID, event.CorrelationID, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	if result.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		return true, nil
	}

	q := r.queries.WithTx(tx)
	orderRow, err := q.GetOrder(ctx, event.AggregateID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	sagaRow, err := q.GetSagaInstanceByOrderID(ctx, event.AggregateID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	payload := paymentEventPayload{}
	if event.Payload != nil {
		raw, marshalErr := json.Marshal(event.Payload)
		if marshalErr != nil {
			span.RecordError(marshalErr)
			span.SetStatus(codes.Error, marshalErr.Error())
			return false, marshalErr
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	}

	switch event.EventType {
	case "PaymentCreated":
		if orderRow.Status == model.OrderStatusStockReserved && payload.PaymentID != "" {
			if err := q.UpdateOrderStatusWithPayment(ctx, sqlc.UpdateOrderStatusWithPaymentParams{
				ID:        event.AggregateID,
				Status:    model.OrderStatusWaitingPayment,
				PaymentID: nullableText(payload.PaymentID),
			}); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return false, err
			}
			if err := q.UpdateSagaInstance(ctx, sqlc.UpdateSagaInstanceParams{
				OrderID:     event.AggregateID,
				Status:      model.SagaStatusPaymentCreated,
				CurrentStep: model.SagaStepCreatePayment,
				Column4:     false,
			}); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return false, err
			}
		}
	case "PaymentSucceeded":
		if isOrderTerminal(orderRow.Status) {
			break
		}
		if err := q.UpdateOrderStatusWithPayment(ctx, sqlc.UpdateOrderStatusWithPaymentParams{
			ID:        event.AggregateID,
			Status:    model.OrderStatusConfirmed,
			PaymentID: nullableText(firstNonEmpty(payload.PaymentID, orderRow.PaymentID.String)),
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		if err := q.UpdateSagaInstance(ctx, sqlc.UpdateSagaInstanceParams{
			OrderID:     event.AggregateID,
			Status:      model.SagaStatusCompleted,
			CurrentStep: model.SagaStepCreatePayment,
			Column4:     true,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		if sagaRow.ID != "" {
			if err := createSagaStep(ctx, q, model.SagaStep{
				ID:             newID("saga_step"),
				SagaID:         sagaRow.ID,
				StepName:       model.SagaStepCreatePayment,
				Status:         model.SagaStepStatusSucceeded,
				IdempotencyKey: "payment-succeeded:" + event.AggregateID,
			}); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return false, err
			}
		}
		if err := createOutboxEvent(ctx, q, paymentSucceededOrderEvent(event, payload)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	case "PaymentFailed":
		if isOrderTerminal(orderRow.Status) {
			break
		}
		if err := q.UpdateOrderStatusWithPayment(ctx, sqlc.UpdateOrderStatusWithPaymentParams{
			ID:        event.AggregateID,
			Status:    model.OrderStatusCancelled,
			PaymentID: nullableText(firstNonEmpty(payload.PaymentID, orderRow.PaymentID.String)),
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		if err := q.UpdateSagaInstance(ctx, sqlc.UpdateSagaInstanceParams{
			OrderID:     event.AggregateID,
			Status:      model.SagaStatusCompensated,
			CurrentStep: model.SagaStepReleaseStock,
			Column4:     true,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		if sagaRow.ID != "" {
			if err := createSagaStep(ctx, q, model.SagaStep{
				ID:             newID("saga_step"),
				SagaID:         sagaRow.ID,
				StepName:       model.SagaStepReleaseStock,
				Status:         model.SagaStepStatusSucceeded,
				IdempotencyKey: "payment-failed:" + event.AggregateID,
			}); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return false, err
			}
		}
		if err := createOutboxEvent(ctx, q, paymentFailedOrderEvent(event, payload)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	return false, nil
}

func (r *OrderRepository) RecordTransition(ctx context.Context, transition model.SagaTransition) error {
	ctx, span := orderRepositoryTracer.Start(ctx, "postgres.OrderRepository.RecordTransition")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("order_id", transition.OrderID),
		attribute.String("saga_step", transition.SagaCurrentStep),
	)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer rollback(ctx, tx)

	q := r.queries.WithTx(tx)
	if transition.OrderStatus != "" {
		if transition.PaymentID != "" {
			err = q.UpdateOrderStatusWithPayment(ctx, sqlc.UpdateOrderStatusWithPaymentParams{
				ID:        transition.OrderID,
				Status:    transition.OrderStatus,
				PaymentID: nullableText(transition.PaymentID),
			})
		} else {
			err = q.UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
				ID:     transition.OrderID,
				Status: transition.OrderStatus,
			})
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if transition.SagaStatus != "" {
		if err := q.UpdateSagaInstance(ctx, sqlc.UpdateSagaInstanceParams{
			OrderID:     transition.OrderID,
			Status:      transition.SagaStatus,
			CurrentStep: transition.SagaCurrentStep,
			Column4:     transition.CompleteSaga,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	if transition.Step.ID != "" {
		if err := createSagaStep(ctx, q, transition.Step); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	if transition.Event.ID != "" {
		if err := createOutboxEvent(ctx, q, transition.Event); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func createSagaStep(ctx context.Context, q *sqlc.Queries, step model.SagaStep) error {
	return q.CreateSagaStep(ctx, sqlc.CreateSagaStepParams{
		ID:             step.ID,
		SagaID:         step.SagaID,
		StepName:       step.StepName,
		Status:         step.Status,
		IdempotencyKey: step.IdempotencyKey,
		ErrorMessage:   nullableText(step.ErrorMessage),
	})
}

func createOutboxEvent(ctx context.Context, q *sqlc.Queries, event model.OutboxEvent) error {
	return q.CreateOutboxEvent(ctx, sqlc.CreateOutboxEventParams{
		ID:            event.ID,
		AggregateID:   event.AggregateID,
		AggregateType: event.AggregateType,
		EventType:     event.EventType,
		CorrelationID: event.CorrelationID,
		CausationID:   nullableText(event.CausationID),
		Traceparent:   nullableText(event.Traceparent),
		Payload:       event.Payload,
		Status:        event.Status,
	})
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func nullableText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func numeric(value float64) pgtype.Numeric {
	var result pgtype.Numeric
	_ = result.Scan(strconv.FormatFloat(value, 'f', -1, 64))
	return result
}

func numericFloat(value pgtype.Numeric) float64 {
	floatValue, err := value.Float64Value()
	if err != nil || !floatValue.Valid {
		return 0
	}
	return floatValue.Float64
}

func orderFromRow(row sqlc.Order, itemRows []sqlc.OrderItem) model.Order {
	items := make([]model.OrderItem, 0, len(itemRows))
	for _, item := range itemRows {
		items = append(items, model.OrderItem{
			ID:          item.ID,
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    numericFloat(item.Quantity),
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}

	return model.Order{
		ID:              row.ID,
		CustomerName:    row.CustomerName,
		CustomerPhone:   row.CustomerPhone,
		CustomerAddress: row.CustomerAddress.String,
		Status:          row.Status,
		TotalAmount:     row.TotalAmount,
		PaymentID:       row.PaymentID.String,
		CorrelationID:   row.CorrelationID,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		Items:           items,
	}
}

type paymentEventPayload struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

func canCancelOrder(status string) bool {
	switch status {
	case model.OrderStatusPending, model.OrderStatusStockReserved, model.OrderStatusWaitingPayment:
		return true
	default:
		return false
	}
}

func isOrderTerminal(status string) bool {
	switch status {
	case model.OrderStatusConfirmed, model.OrderStatusCancelled, model.OrderStatusRejected:
		return true
	default:
		return false
	}
}

func orderCancelledEvent(command model.CancelOrderCommand) model.OutboxEvent {
	payload, _ := json.Marshal(map[string]any{
		"order_id": command.OrderID,
		"status":   model.OrderStatusCancelled,
		"reason":   command.Reason,
	})
	return model.OutboxEvent{
		ID:            newID("evt"),
		AggregateID:   command.OrderID,
		AggregateType: "order",
		EventType:     "OrderCancelled",
		CorrelationID: command.CorrelationID,
		CausationID:   command.CausationID,
		Payload:       payload,
		Status:        model.OutboxStatusPending,
	}
}

func paymentSucceededOrderEvent(event messaging.EventEnvelope, payload paymentEventPayload) model.OutboxEvent {
	body, _ := json.Marshal(map[string]any{
		"order_id":   event.AggregateID,
		"status":     model.OrderStatusConfirmed,
		"payment_id": payload.PaymentID,
	})
	return model.OutboxEvent{
		ID:            newID("evt"),
		AggregateID:   event.AggregateID,
		AggregateType: "order",
		EventType:     "OrderConfirmed",
		CorrelationID: event.CorrelationID,
		CausationID:   event.EventID,
		Traceparent:   event.Traceparent,
		Payload:       body,
		Status:        model.OutboxStatusPending,
	}
}

func paymentFailedOrderEvent(event messaging.EventEnvelope, payload paymentEventPayload) model.OutboxEvent {
	body, _ := json.Marshal(map[string]any{
		"order_id":   event.AggregateID,
		"status":     model.OrderStatusCancelled,
		"payment_id": payload.PaymentID,
		"reason":     firstNonEmpty(payload.Reason, "payment failed"),
	})
	return model.OutboxEvent{
		ID:            newID("evt"),
		AggregateID:   event.AggregateID,
		AggregateType: "order",
		EventType:     "OrderCancelled",
		CorrelationID: event.CorrelationID,
		CausationID:   event.EventID,
		Traceparent:   event.Traceparent,
		Payload:       body,
		Status:        model.OutboxStatusPending,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func newID(prefix string) string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return prefix + "_fallback"
	}
	return prefix + "_" + hex.EncodeToString(raw)
}
