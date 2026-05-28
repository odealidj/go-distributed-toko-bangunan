package usecase

import (
	"context"
	"testing"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
)

func TestListProductsUsesCache(t *testing.T) {
	repository := &fakeRepository{}
	cache := &fakeCache{
		list: port.ProductList{
			Items: []model.Product{{ID: "prod_semen_50kg", Name: "Semen Portland 50kg"}},
			Total: 1,
		},
		listHit: true,
	}
	uc := NewCatalog(repository, cache)

	result, err := uc.ListProducts(context.Background(), model.ProductFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repository.listCalled {
		t.Fatal("expected repository not called when cache hit")
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestReserveStockNormalizesItemsAndInvalidatesCache(t *testing.T) {
	repository := &fakeRepository{}
	cache := &fakeCache{}
	uc := NewCatalog(repository, cache)

	_, err := uc.ReserveStock(context.Background(), model.ReserveStockCommand{
		OrderID:        "ord_1",
		IdempotencyKey: "idem_1",
		Items: []model.OrderItemInput{
			{ProductID: "prod_besi_10mm", Quantity: 1},
			{ProductID: "prod_besi_10mm", Quantity: 2},
			{ProductID: "", Quantity: 10},
			{ProductID: "prod_semen_50kg", Quantity: 0},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repository.reserveCommand.Items) != 1 {
		t.Fatalf("expected one normalized item, got %d", len(repository.reserveCommand.Items))
	}
	if repository.reserveCommand.Items[0].Quantity != 3 {
		t.Fatalf("expected merged quantity 3, got %v", repository.reserveCommand.Items[0].Quantity)
	}
	if !cache.productDeleted["prod_besi_10mm"] {
		t.Fatal("expected product cache deleted")
	}
	if !cache.listsDeleted {
		t.Fatal("expected list cache deleted")
	}
}

type fakeRepository struct {
	listCalled     bool
	reserveCommand model.ReserveStockCommand
}

func (r *fakeRepository) Ping(context.Context) error { return nil }

func (r *fakeRepository) ListProducts(context.Context, model.ProductFilter) (port.ProductList, error) {
	r.listCalled = true
	return port.ProductList{}, nil
}

func (r *fakeRepository) GetProduct(context.Context, string) (model.Product, error) {
	return model.Product{}, model.ErrProductNotFound
}

func (r *fakeRepository) ValidateProducts(context.Context, []model.OrderItemInput) ([]model.ValidatedItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeRepository) ReserveStock(_ context.Context, command model.ReserveStockCommand) (model.StockReservation, error) {
	r.reserveCommand = command
	return model.StockReservation{
		ID:     "res_1",
		Status: model.ReservationStatusReserved,
		Items:  command.Items,
	}, nil
}

func (r *fakeRepository) ReleaseStock(context.Context, string) (model.StockReservation, error) {
	return model.StockReservation{}, nil
}

func (r *fakeRepository) CommitStock(context.Context, string) (model.StockReservation, error) {
	return model.StockReservation{}, nil
}

type fakeCache struct {
	list           port.ProductList
	listHit        bool
	productDeleted map[string]bool
	listsDeleted   bool
}

func (c *fakeCache) GetProduct(context.Context, string) (model.Product, bool) {
	return model.Product{}, false
}

func (c *fakeCache) SetProduct(context.Context, model.Product) {}

func (c *fakeCache) GetProductList(context.Context, model.ProductFilter) (port.ProductList, bool) {
	return c.list, c.listHit
}

func (c *fakeCache) SetProductList(context.Context, model.ProductFilter, port.ProductList) {}

func (c *fakeCache) DeleteProduct(_ context.Context, id string) {
	if c.productDeleted == nil {
		c.productDeleted = map[string]bool{}
	}
	c.productDeleted[id] = true
}

func (c *fakeCache) DeleteProductLists(context.Context) {
	c.listsDeleted = true
}
