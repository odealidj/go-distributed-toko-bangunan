package port

import (
	"context"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

type ProductList struct {
	Items []model.Product
	Total int
}

type CatalogRepository interface {
	Ping(ctx context.Context) error
	ListProducts(ctx context.Context, filter model.ProductFilter) (ProductList, error)
	GetProduct(ctx context.Context, id string) (model.Product, error)
	ValidateProducts(ctx context.Context, items []model.OrderItemInput) ([]model.ValidatedItem, int64, error)
	ReserveStock(ctx context.Context, command model.ReserveStockCommand) (model.StockReservation, error)
	ReleaseStock(ctx context.Context, orderID string) (model.StockReservation, error)
	CommitStock(ctx context.Context, orderID string) (model.StockReservation, error)
	ProcessOrderEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error)
}

type ProductCache interface {
	GetProduct(ctx context.Context, id string) (model.Product, bool)
	SetProduct(ctx context.Context, product model.Product)
	GetProductList(ctx context.Context, filter model.ProductFilter) (ProductList, bool)
	SetProductList(ctx context.Context, filter model.ProductFilter, list ProductList)
	DeleteProduct(ctx context.Context, id string)
	DeleteProductLists(ctx context.Context)
}
