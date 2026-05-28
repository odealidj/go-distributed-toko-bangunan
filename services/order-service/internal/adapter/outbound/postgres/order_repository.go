package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
)

type OrderRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *OrderRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *OrderRepository) CreateCheckout(ctx context.Context, order model.Order, saga model.SagaInstance, step model.SagaStep, event model.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
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
		return err
	}
	if err := createSagaStep(ctx, q, step); err != nil {
		return err
	}
	if err := createOutboxEvent(ctx, q, event); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (model.Order, error) {
	row, err := r.queries.GetOrder(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, model.ErrOrderNotFound
	}
	if err != nil {
		return model.Order{}, err
	}
	items, err := r.queries.ListOrderItems(ctx, orderID)
	if err != nil {
		return model.Order{}, err
	}
	return orderFromRow(row, items), nil
}

func (r *OrderRepository) RecordTransition(ctx context.Context, transition model.SagaTransition) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
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
			return err
		}
	}
	if transition.Step.ID != "" {
		if err := createSagaStep(ctx, q, transition.Step); err != nil {
			return err
		}
	}
	if transition.Event.ID != "" {
		if err := createOutboxEvent(ctx, q, transition.Event); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
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
