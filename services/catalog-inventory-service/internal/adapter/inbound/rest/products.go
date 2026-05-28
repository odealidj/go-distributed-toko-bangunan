package rest

import (
	"errors"
	"net/http"
	"strconv"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

type productResponse struct {
	ID            string  `json:"id"`
	CategoryID    string  `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	SKU           string  `json:"sku"`
	Name          string  `json:"name"`
	Brand         string  `json:"brand"`
	Unit          string  `json:"unit"`
	Price         int64   `json:"price"`
	WeightKG      float64 `json:"weight_kg"`
	RequiresTruck bool    `json:"requires_truck"`
	AvailableQty  float64 `json:"available_qty"`
}

func listProductsHandler(catalog *usecase.CatalogUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		query := ctx.Query()
		page := parsePositiveInt(query.Get("page"), 1)
		perPage := parsePositiveInt(query.Get("per_page"), 10)

		result, err := catalog.ListProducts(ctx, model.ProductFilter{
			CategoryID: query.Get("category_id"),
			Search:     query.Get("search"),
			Page:       page,
			PerPage:    perPage,
		})
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "CATALOG_QUERY_FAILED", "Gagal mengambil daftar produk.")
		}

		products := make([]productResponse, 0, len(result.Items))
		for _, product := range result.Items {
			products = append(products, newProductResponse(product))
		}
		return response.JSONPage(ctx, http.StatusOK, products, response.NewPagination(page, perPage, result.Total))
	}
}

func getProductHandler(catalog *usecase.CatalogUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		id := ctx.Vars().Get("id")
		product, err := catalog.GetProduct(ctx, id)
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_PRODUCT_ID", "ID produk tidak valid.")
		}
		if errors.Is(err, model.ErrProductNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Produk tidak ditemukan.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "CATALOG_QUERY_FAILED", "Gagal mengambil produk.")
		}
		return response.JSON(ctx, http.StatusOK, newProductResponse(product))
	}
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func newProductResponse(product model.Product) productResponse {
	return productResponse{
		ID:            product.ID,
		CategoryID:    product.CategoryID,
		CategoryName:  product.CategoryName,
		SKU:           product.SKU,
		Name:          product.Name,
		Brand:         product.Brand,
		Unit:          product.Unit,
		Price:         product.Price,
		WeightKG:      product.WeightKG,
		RequiresTruck: product.RequiresTruck,
		AvailableQty:  product.AvailableQty,
	}
}
