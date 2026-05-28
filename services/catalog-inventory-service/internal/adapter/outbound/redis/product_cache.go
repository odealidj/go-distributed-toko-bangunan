package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	goredis "github.com/redis/go-redis/v9"
)

const (
	productTTL = 10 * time.Minute
	listTTL    = 2 * time.Minute
)

type ProductCache struct {
	client *goredis.Client
}

func NewProductCache(client *goredis.Client) *ProductCache {
	return &ProductCache{client: client}
}

func (c *ProductCache) GetProduct(ctx context.Context, id string) (model.Product, bool) {
	var product model.Product
	if c == nil || c.client == nil {
		return product, false
	}
	raw, err := c.client.Get(ctx, productKey(id)).Bytes()
	if err != nil {
		return product, false
	}
	if err := json.Unmarshal(raw, &product); err != nil {
		return model.Product{}, false
	}
	return product, true
}

func (c *ProductCache) SetProduct(ctx context.Context, product model.Product) {
	if c == nil || c.client == nil {
		return
	}
	raw, err := json.Marshal(product)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, productKey(product.ID), raw, productTTL).Err()
}

func (c *ProductCache) GetProductList(ctx context.Context, filter model.ProductFilter) (port.ProductList, bool) {
	var list port.ProductList
	if c == nil || c.client == nil {
		return list, false
	}
	raw, err := c.client.Get(ctx, listKey(filter)).Bytes()
	if err != nil {
		return list, false
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return port.ProductList{}, false
	}
	return list, true
}

func (c *ProductCache) SetProductList(ctx context.Context, filter model.ProductFilter, list port.ProductList) {
	if c == nil || c.client == nil {
		return
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, listKey(filter), raw, listTTL).Err()
}

func (c *ProductCache) DeleteProduct(ctx context.Context, id string) {
	if c == nil || c.client == nil {
		return
	}
	_ = c.client.Del(ctx, productKey(id)).Err()
}

func (c *ProductCache) DeleteProductLists(ctx context.Context) {
	if c == nil || c.client == nil {
		return
	}
	iter := c.client.Scan(ctx, 0, "catalog-inventory-service:products:list:*", 100).Iterator()
	for iter.Next(ctx) {
		_ = c.client.Del(ctx, iter.Val()).Err()
	}
}

func productKey(id string) string {
	return "catalog-inventory-service:product:" + id
}

func listKey(filter model.ProductFilter) string {
	return fmt.Sprintf(
		"catalog-inventory-service:products:list:category:%s:search:%s:page:%d:per_page:%d",
		filter.CategoryID,
		filter.Search,
		filter.Page,
		filter.PerPage,
	)
}
