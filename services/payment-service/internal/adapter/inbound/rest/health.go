package rest

import (
	"errors"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

func RegisterRoutes(server *khttp.Server, cfg config.ServiceConfig, payment *usecase.PaymentUseCase) {
	router := server.Route("/")
	router.GET("/healthz", healthHandler(cfg))
	router.GET("/readyz", readyHandler(cfg, payment))
	router.GET("/payments/{id}", getPaymentHandler(payment))
}

func healthHandler(cfg config.ServiceConfig) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		return response.JSON(ctx, http.StatusOK, healthResponse{
			Service: cfg.ServiceName,
			Status:  "ok",
		})
	}
}

func readyHandler(cfg config.ServiceConfig, payment *usecase.PaymentUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		databaseStatus := "ok"
		statusCode := http.StatusOK
		status := "ready"
		if err := payment.Ping(ctx); err != nil {
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
				Redis:    "not_configured",
			},
		})
	}
}

func getPaymentHandler(payment *usecase.PaymentUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		result, err := payment.GetPaymentByID(ctx, ctx.Vars().Get("id"))
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_PAYMENT_ID", "ID payment tidak valid.")
		}
		if errors.Is(err, model.ErrPaymentNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "PAYMENT_NOT_FOUND", "Payment tidak ditemukan.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "PAYMENT_QUERY_FAILED", "Gagal mengambil payment.")
		}
		return response.JSON(ctx, http.StatusOK, paymentResponse{
			ID:             result.ID,
			OrderID:        result.OrderID,
			Amount:         result.Amount,
			Status:         result.Status,
			PaymentMode:    result.PaymentMode,
			IdempotencyKey: result.IdempotencyKey,
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

type paymentResponse struct {
	ID             string `json:"id"`
	OrderID        string `json:"order_id"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	PaymentMode    string `json:"payment_mode"`
	IdempotencyKey string `json:"idempotency_key"`
}
