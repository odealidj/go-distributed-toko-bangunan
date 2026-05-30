package rest

import (
	"errors"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

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

func succeedPaymentHandler(payment *usecase.PaymentUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		result, err := payment.SucceedPayment(ctx, model.CompletePaymentCommand{
			PaymentID:     ctx.Vars().Get("id"),
			Reason:        "manual_success",
			CorrelationID: correlationID(ctx),
			CausationID:   requestID(ctx),
		})
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_PAYMENT_ID", "ID payment tidak valid.")
		}
		if errors.Is(err, model.ErrPaymentNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "PAYMENT_NOT_FOUND", "Payment tidak ditemukan.")
		}
		if errors.Is(err, model.ErrPaymentConflict) {
			return response.JSONError(ctx, http.StatusConflict, "PAYMENT_STATE_CONFLICT", "Payment tidak bisa diubah ke success dari status saat ini.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "PAYMENT_UPDATE_FAILED", "Gagal mengubah payment ke success.")
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

func failPaymentHandler(payment *usecase.PaymentUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		result, err := payment.FailPayment(ctx, model.CompletePaymentCommand{
			PaymentID:     ctx.Vars().Get("id"),
			Reason:        "manual_failure",
			CorrelationID: correlationID(ctx),
			CausationID:   requestID(ctx),
		})
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_PAYMENT_ID", "ID payment tidak valid.")
		}
		if errors.Is(err, model.ErrPaymentNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "PAYMENT_NOT_FOUND", "Payment tidak ditemukan.")
		}
		if errors.Is(err, model.ErrPaymentConflict) {
			return response.JSONError(ctx, http.StatusConflict, "PAYMENT_STATE_CONFLICT", "Payment tidak bisa diubah ke failure dari status saat ini.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "PAYMENT_UPDATE_FAILED", "Gagal mengubah payment ke failure.")
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

type paymentResponse struct {
	ID             string `json:"id"`
	OrderID        string `json:"order_id"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	PaymentMode    string `json:"payment_mode"`
	IdempotencyKey string `json:"idempotency_key"`
}

func correlationID(ctx khttp.Context) string {
	value := ctx.Request().Header.Get("X-Correlation-ID")
	if value == "" {
		value = ctx.Request().Header.Get("X-Correlation-Id")
	}
	return value
}

func requestID(ctx khttp.Context) string {
	value := ctx.Request().Header.Get("X-Request-ID")
	if value == "" {
		value = ctx.Request().Header.Get("X-Request-Id")
	}
	return value
}
