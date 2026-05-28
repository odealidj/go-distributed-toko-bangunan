package grpc

import (
	"context"
	"errors"
	"time"

	inventoryv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/inventory/v1"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryClient struct {
	client  inventoryv1.InventoryServiceClient
	timeout time.Duration
	retries int
}

func NewInventoryClient(client inventoryv1.InventoryServiceClient) *InventoryClient {
	return &InventoryClient{
		client:  client,
		timeout: 2 * time.Second,
		retries: 1,
	}
}

func (c *InventoryClient) ValidateProducts(ctx context.Context, items []model.OrderItemInput, correlationID, causationID string) ([]model.ValidatedOrderItem, int64, error) {
	var response *inventoryv1.ValidateProductsResponse
	err := c.call(ctx, func(callCtx context.Context) error {
		var err error
		response, err = c.client.ValidateProducts(callCtx, &inventoryv1.ValidateProductsRequest{
			Metadata: metadata(correlationID, causationID, ""),
			Items:    inventoryItems(items),
		})
		return err
	})
	if err != nil {
		return nil, 0, inventoryError(err)
	}

	result := make([]model.ValidatedOrderItem, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		result = append(result, model.ValidatedOrderItem{
			ProductID:   item.GetProductId(),
			ProductName: item.GetProductName(),
			Unit:        item.GetUnit(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   item.GetUnitPrice(),
			LineTotal:   item.GetLineTotal(),
		})
	}
	return result, response.GetTotalAmount(), nil
}

func (c *InventoryClient) ReserveStock(ctx context.Context, orderID string, items []model.OrderItemInput, correlationID, causationID, idempotencyKey string) error {
	var response *inventoryv1.ReserveStockResponse
	err := c.call(ctx, func(callCtx context.Context) error {
		var err error
		response, err = c.client.ReserveStock(callCtx, &inventoryv1.ReserveStockRequest{
			Metadata: metadata(correlationID, causationID, idempotencyKey),
			OrderId:  orderID,
			Items:    inventoryItems(items),
		})
		return err
	})
	if err != nil {
		return inventoryError(err)
	}
	if response.GetStatus() != "RESERVED" {
		return model.ErrInsufficientStock
	}
	return nil
}

func (c *InventoryClient) ReleaseStock(ctx context.Context, orderID, correlationID, causationID, idempotencyKey string) error {
	return inventoryError(c.call(ctx, func(callCtx context.Context) error {
		_, err := c.client.ReleaseStock(callCtx, &inventoryv1.ReleaseStockRequest{
			Metadata: metadata(correlationID, causationID, idempotencyKey),
			OrderId:  orderID,
		})
		return err
	}))
}

func (c *InventoryClient) CommitStock(ctx context.Context, orderID, correlationID, causationID, idempotencyKey string) error {
	return inventoryError(c.call(ctx, func(callCtx context.Context) error {
		_, err := c.client.CommitStock(callCtx, &inventoryv1.CommitStockRequest{
			Metadata: metadata(correlationID, causationID, idempotencyKey),
			OrderId:  orderID,
		})
		return err
	}))
}

func (c *InventoryClient) call(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, c.timeout)
		lastErr = fn(callCtx)
		cancel()
		if lastErr == nil || !isTransient(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func inventoryError(err error) error {
	if err == nil {
		return nil
	}
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument:
		return model.ErrInvalidInput
	case codes.NotFound:
		return model.ErrProductNotFound
	case codes.FailedPrecondition:
		return model.ErrInsufficientStock
	default:
		return err
	}
}

func inventoryItems(items []model.OrderItemInput) []*inventoryv1.OrderItemInput {
	result := make([]*inventoryv1.OrderItemInput, 0, len(items))
	for _, item := range items {
		result = append(result, &inventoryv1.OrderItemInput{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}
	return result
}

func metadata(correlationID, causationID, idempotencyKey string) *inventoryv1.RequestMetadata {
	return &inventoryv1.RequestMetadata{
		CorrelationId:  correlationID,
		CausationId:    causationID,
		IdempotencyKey: idempotencyKey,
	}
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
