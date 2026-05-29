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
		return response.JSON(ctx, http.StatusOK, healthResponse{
			Service: cfg.ServiceName,
			Status:  "ok",
		})
	}
}

func readyHandler(cfg config.ServiceConfig, catalog *usecase.CatalogUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		databaseStatus := "ok"
		statusCode := http.StatusOK
		status := "ready"
		if err := catalog.Ping(ctx); err != nil {
			databaseStatus = "unavailable"
			statusCode = http.StatusServiceUnavailable
			status = "not_ready"
		}

		return response.JSON(ctx, statusCode, readinessResponse{
			Service: cfg.ServiceName,
			Status:  status,
			Checks: readinessChecks{
				Database: databaseStatus,
				Kafka:    "not_configured",
				Redis:    "optional_cache",
			},
		})
	}
}

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type readinessResponse struct {
	Service string          `json:"service"`
	Status  string          `json:"status"`
	Checks  readinessChecks `json:"checks"`
}

type readinessChecks struct {
	Database string `json:"database"`
	Kafka    string `json:"kafka"`
	Redis    string `json:"redis"`
}
