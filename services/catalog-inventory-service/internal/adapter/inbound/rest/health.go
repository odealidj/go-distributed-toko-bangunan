package rest

import (
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

func RegisterRoutes(server *khttp.Server, cfg config.ServiceConfig, catalog *usecase.CatalogUseCase) {
	router := server.Route("/")
	router.GET("/healthz", healthHandler(cfg))
	router.GET("/readyz", readyHandler(cfg, catalog))
	router.GET("/products", listProductsHandler(catalog))
	router.GET("/products/{id}", getProductHandler(catalog))
}

func healthHandler(cfg config.ServiceConfig) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		return response.JSON(ctx, http.StatusOK, map[string]any{
			"service": cfg.ServiceName,
			"status":  "ok",
		})
	}
}

func readyHandler(cfg config.ServiceConfig, catalog *usecase.CatalogUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		databaseStatus := "ok"
		if err := catalog.Ping(ctx); err != nil {
			databaseStatus = "unavailable"
		}

		return response.JSON(ctx, http.StatusOK, map[string]any{
			"service": cfg.ServiceName,
			"status":  "ready",
			"checks": map[string]string{
				"database": databaseStatus,
				"kafka":    "not_configured",
				"redis":    "optional_cache",
			},
		})
	}
}
