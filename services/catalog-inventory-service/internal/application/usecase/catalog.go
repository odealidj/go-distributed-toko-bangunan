package usecase

import (
	"context"
	"sort"
	"strings"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type CatalogUseCase struct {
	repository port.CatalogRepository
	cache      port.ProductCache
}

func NewCatalog(repository port.CatalogRepository, cache port.ProductCache) *CatalogUseCase {
	return &CatalogUseCase{
		repository: repository,
		cache:      cache,
	}
}

func (u *CatalogUseCase) Ping(ctx context.Context) error {
	return u.repository.Ping(ctx)
}

func (u *CatalogUseCase) ListProducts(ctx context.Context, filter model.ProductFilter) (port.ProductList, error) {
	filter = normalizeFilter(filter)
	if u.cache != nil {
		if cached, ok := u.cache.GetProductList(ctx, filter); ok {
			return cached, nil
		}
	}

	result, err := u.repository.ListProducts(ctx, filter)
	if err != nil {
		return port.ProductList{}, err
	}
	if u.cache != nil {
		u.cache.SetProductList(ctx, filter, result)
	}
	return result, nil
}

func (u *CatalogUseCase) GetProduct(ctx context.Context, id string) (model.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.Product{}, model.ErrInvalidInput
	}
	if u.cache != nil {
		if cached, ok := u.cache.GetProduct(ctx, id); ok {
			return cached, nil
		}
	}

	product, err := u.repository.GetProduct(ctx, id)
	if err != nil {
		return model.Product{}, err
	}
	if u.cache != nil {
		u.cache.SetProduct(ctx, product)
	}
	return product, nil
}

func (u *CatalogUseCase) ValidateProducts(ctx context.Context, items []model.OrderItemInput) ([]model.ValidatedItem, int64, error) {
	items = normalizeItems(items)
	if len(items) == 0 {
		return nil, 0, model.ErrInvalidInput
	}
	return u.repository.ValidateProducts(ctx, items)
}

func (u *CatalogUseCase) ReserveStock(ctx context.Context, command model.ReserveStockCommand) (model.StockReservation, error) {
	command.OrderID = strings.TrimSpace(command.OrderID)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.Items = normalizeItems(command.Items)
	if command.OrderID == "" || command.IdempotencyKey == "" || len(command.Items) == 0 {
		return model.StockReservation{}, model.ErrInvalidInput
	}

	reservation, err := u.repository.ReserveStock(ctx, command)
	if err == nil && u.cache != nil {
		u.invalidateStockCache(ctx, command.Items)
	}
	return reservation, err
}

func (u *CatalogUseCase) ReleaseStock(ctx context.Context, orderID string) (model.StockReservation, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return model.StockReservation{}, model.ErrInvalidInput
	}
	reservation, err := u.repository.ReleaseStock(ctx, orderID)
	if err == nil && u.cache != nil {
		u.invalidateStockCache(ctx, reservation.Items)
	}
	return reservation, err
}

func (u *CatalogUseCase) CommitStock(ctx context.Context, orderID string) (model.StockReservation, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return model.StockReservation{}, model.ErrInvalidInput
	}
	reservation, err := u.repository.CommitStock(ctx, orderID)
	if err == nil && u.cache != nil {
		u.invalidateStockCache(ctx, reservation.Items)
	}
	return reservation, err
}

func (u *CatalogUseCase) ProcessOrderEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	return u.repository.ProcessOrderEvent(ctx, event)
}

func normalizeFilter(filter model.ProductFilter) model.ProductFilter {
	filter.CategoryID = strings.TrimSpace(filter.CategoryID)
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}
	return filter
}

func normalizeItems(items []model.OrderItemInput) []model.OrderItemInput {
	merged := make(map[string]float64, len(items))
	for _, item := range items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" || item.Quantity <= 0 {
			continue
		}
		merged[productID] += item.Quantity
	}

	normalized := make([]model.OrderItemInput, 0, len(merged))
	for productID, quantity := range merged {
		normalized = append(normalized, model.OrderItemInput{
			ProductID: productID,
			Quantity:  quantity,
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].ProductID < normalized[j].ProductID
	})
	return normalized
}

func (u *CatalogUseCase) invalidateStockCache(ctx context.Context, items []model.OrderItemInput) {
	for _, item := range items {
		u.cache.DeleteProduct(ctx, item.ProductID)
	}
	u.cache.DeleteProductLists(ctx)
}
