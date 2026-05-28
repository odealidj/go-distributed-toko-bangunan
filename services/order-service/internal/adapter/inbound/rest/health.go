package rest

import (
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

func RegisterRoutes(server *khttp.Server, cfg config.ServiceConfig, order *usecase.OrderUseCase) {
	router := server.Route("/")
	router.GET("/healthz", healthHandler(cfg))
	router.GET("/readyz", readyHandler(cfg, order))
	router.POST("/orders", createOrderHandler(order))
	router.GET("/orders/{id}", getOrderHandler(order))
}

func healthHandler(cfg config.ServiceConfig) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		return response.JSON(ctx, http.StatusOK, healthResponse{
			Service: cfg.ServiceName,
			Status:  "ok",
		})
	}
}

func readyHandler(cfg config.ServiceConfig, order *usecase.OrderUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		databaseStatus := "ok"
		if err := order.Ping(ctx); err != nil {
			databaseStatus = "unavailable"
		}

		return response.JSON(ctx, http.StatusOK, readinessResponse{
			Service: cfg.ServiceName,
			Status:  "ready",
			Checks: readinessChecks{
				Database: databaseStatus,
				Kafka:    "not_configured",
				Redis:    "not_configured",
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
