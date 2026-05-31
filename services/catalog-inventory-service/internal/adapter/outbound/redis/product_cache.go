package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	productTTL = 10 * time.Minute
	listTTL    = 2 * time.Minute
)

type ProductCache struct {
	client *goredis.Client
}

var productCacheTracer = otel.Tracer("catalog-inventory-service/redis")

func NewProductCache(client *goredis.Client) *ProductCache {
	return &ProductCache{client: client}
}

func (c *ProductCache) GetProduct(ctx context.Context, id string) (model.Product, bool) {
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.GetProduct")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"), attribute.String("product_id", id))
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
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.SetProduct")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"), attribute.String("product_id", product.ID))
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
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.GetProductList")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"))
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
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.SetProductList")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"))
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
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.DeleteProduct")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"), attribute.String("product_id", id))
	if c == nil || c.client == nil {
		return
	}
	_ = c.client.Del(ctx, productKey(id)).Err()
}

func (c *ProductCache) DeleteProductLists(ctx context.Context) {
	ctx, span := productCacheTracer.Start(ctx, "redis.ProductCache.DeleteProductLists")
	defer span.End()
	span.SetAttributes(attribute.String("cache.system", "redis"))
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
