package grpc

import (
	"context"
	"errors"

	"github.com/odealidj/go-distributed-toko-bangunan/proto/inventory/v1"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	catalog *usecase.CatalogUseCase
}

func NewInventoryServer(catalog *usecase.CatalogUseCase) *InventoryServer {
	return &InventoryServer{catalog: catalog}
}

func (s *InventoryServer) GetProduct(ctx context.Context, req *inventoryv1.GetProductRequest) (*inventoryv1.GetProductResponse, error) {
	product, err := s.catalog.GetProduct(ctx, req.GetProductId())
	if errors.Is(err, model.ErrProductNotFound) {
		return nil, status.Error(codes.NotFound, "produk tidak ditemukan")
	}
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "product_id wajib diisi")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &inventoryv1.GetProductResponse{
		Product: productSnapshot(product),
	}, nil
}

func (s *InventoryServer) ValidateProducts(ctx context.Context, req *inventoryv1.ValidateProductsRequest) (*inventoryv1.ValidateProductsResponse, error) {
	items, total, err := s.catalog.ValidateProducts(ctx, orderItems(req.GetItems()))
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "items wajib diisi")
	}
	if errors.Is(err, model.ErrProductNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, model.ErrInsufficientStock) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	responseItems := make([]*inventoryv1.ValidatedOrderItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, &inventoryv1.ValidatedOrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}
	return &inventoryv1.ValidateProductsResponse{
		Items:       responseItems,
		TotalAmount: total,
	}, nil
}

func (s *InventoryServer) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	reservation, err := s.catalog.ReserveStock(ctx, model.ReserveStockCommand{
		OrderID:        req.GetOrderId(),
		IdempotencyKey: req.GetMetadata().GetIdempotencyKey(),
		Items:          orderItems(req.GetItems()),
	})
	if errors.Is(err, model.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "order_id, idempotency_key, dan items wajib diisi")
	}
	if errors.Is(err, model.ErrInsufficientStock) {
		return &inventoryv1.ReserveStockResponse{
			Status:        model.ReservationStatusFailed,
			FailureReason: err.Error(),
		}, nil
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &inventoryv1.ReserveStockResponse{
		ReservationId: reservation.ID,
		Status:        reservation.Status,
	}, nil
}

func (s *InventoryServer) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.ReleaseStockResponse, error) {
	reservation, err := s.catalog.ReleaseStock(ctx, req.GetOrderId())
	if err != nil {
		return stockTransitionError(err)
	}
	return &inventoryv1.ReleaseStockResponse{
		ReservationId: reservation.ID,
		Status:        reservation.Status,
	}, nil
}

func (s *InventoryServer) CommitStock(ctx context.Context, req *inventoryv1.CommitStockRequest) (*inventoryv1.CommitStockResponse, error) {
	reservation, err := s.catalog.CommitStock(ctx, req.GetOrderId())
	if err != nil {
		return nil, stockTransitionStatusError(err)
	}
	return &inventoryv1.CommitStockResponse{
		ReservationId: reservation.ID,
		Status:        reservation.Status,
	}, nil
}

func stockTransitionError(err error) (*inventoryv1.ReleaseStockResponse, error) {
	return nil, stockTransitionStatusError(err)
}

func stockTransitionStatusError(err error) error {
	if errors.Is(err, model.ErrInvalidInput) {
		return status.Error(codes.InvalidArgument, "order_id wajib diisi")
	}
	if errors.Is(err, model.ErrReservationNotFound) {
		return status.Error(codes.NotFound, "reservasi stock tidak ditemukan")
	}
	return status.Error(codes.Internal, err.Error())
}

func productSnapshot(product model.Product) *inventoryv1.ProductSnapshot {
	return &inventoryv1.ProductSnapshot{
		ProductId:     product.ID,
		Name:          product.Name,
		Category:      product.CategoryName,
		Unit:          product.Unit,
		UnitPrice:     product.Price,
		AvailableQty:  product.AvailableQty,
		WeightKg:      product.WeightKG,
		RequiresTruck: product.RequiresTruck,
	}
}

func orderItems(items []*inventoryv1.OrderItemInput) []model.OrderItemInput {
	result := make([]model.OrderItemInput, 0, len(items))
	for _, item := range items {
		result = append(result, model.OrderItemInput{
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		})
	}
	return result
}
