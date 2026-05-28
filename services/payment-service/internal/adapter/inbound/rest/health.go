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
		return response.JSON(ctx, http.StatusOK, map[string]any{
			"service": cfg.ServiceName,
			"status":  "ok",
		})
	}
}

func readyHandler(cfg config.ServiceConfig, payment *usecase.PaymentUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		databaseStatus := "ok"
		if err := payment.Ping(ctx); err != nil {
			databaseStatus = "unavailable"
		}
		return response.JSON(ctx, http.StatusOK, map[string]any{
			"service": cfg.ServiceName,
			"status":  "ready",
			"checks": map[string]string{
				"database": databaseStatus,
				"kafka":    "not_configured",
				"redis":    "not_configured",
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
		return response.JSON(ctx, http.StatusOK, map[string]any{
			"id":              result.ID,
			"order_id":        result.OrderID,
			"amount":          result.Amount,
			"status":          result.Status,
			"payment_mode":    result.PaymentMode,
			"idempotency_key": result.IdempotencyKey,
		})
	}
}
