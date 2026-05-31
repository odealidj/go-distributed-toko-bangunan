package rest

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/response"
)

type createOrderRequest struct {
	Customer    customerRequest          `json:"customer"`
	PaymentMode string                   `json:"payment_mode"`
	Items       []createOrderItemRequest `json:"items"`
}

type customerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type createOrderItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
}

type orderResponse struct {
	ID            string              `json:"id"`
	Status        string              `json:"status"`
	Customer      customerResponse    `json:"customer"`
	Items         []orderItemResponse `json:"items"`
	TotalAmount   int64               `json:"total_amount"`
	PaymentID     string              `json:"payment_id,omitempty"`
	CorrelationID string              `json:"correlation_id"`
	CreatedAt     string              `json:"created_at,omitempty"`
	UpdatedAt     string              `json:"updated_at,omitempty"`
}

type orderListItemResponse struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	Customer      customerResponse `json:"customer"`
	TotalAmount   int64            `json:"total_amount"`
	PaymentID     string           `json:"payment_id,omitempty"`
	CorrelationID string           `json:"correlation_id"`
	CreatedAt     string           `json:"created_at,omitempty"`
	UpdatedAt     string           `json:"updated_at,omitempty"`
}

type customerResponse struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address,omitempty"`
}

type orderItemResponse struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	LineTotal   int64   `json:"line_total"`
}

func createOrderHandler(order *usecase.OrderUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		var request createOrderRequest
		if err := ctx.Bind(&request); err != nil {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Request body tidak valid.")
		}

		result, err := order.CreateCheckout(ctx, model.CreateCheckoutCommand{
			CustomerName:    request.Customer.Name,
			CustomerPhone:   request.Customer.Phone,
			CustomerAddress: request.Customer.Address,
			PaymentMode:     request.PaymentMode,
			CorrelationID:   correlationID(ctx),
			CausationID:     requestID(ctx),
			Items:           orderItemInputs(request.Items),
		})
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_ORDER_INPUT", "Input order tidak valid.")
		}
		if errors.Is(err, model.ErrProductNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Produk tidak ditemukan.")
		}
		if errors.Is(err, model.ErrInsufficientStock) {
			return response.JSONError(ctx, http.StatusConflict, "INSUFFICIENT_STOCK", "Stock produk tidak cukup.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusBadGateway, "CHECKOUT_FAILED", "Checkout gagal diproses.")
		}

		return response.JSON(ctx, http.StatusCreated, newOrderResponse(result))
	}
}

func getOrderHandler(order *usecase.OrderUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		result, err := order.GetOrder(ctx, ctx.Vars().Get("id"))
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_ORDER_ID", "ID order tidak valid.")
		}
		if errors.Is(err, model.ErrOrderNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND", "Order tidak ditemukan.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "ORDER_QUERY_FAILED", "Gagal mengambil order.")
		}
		return response.JSON(ctx, http.StatusOK, newOrderResponse(result))
	}
}

func listOrdersHandler(order *usecase.OrderUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		page := parsePositiveInt(ctx.Query().Get("page"), 1)
		perPage := parsePositiveInt(ctx.Query().Get("per_page"), 10)
		status := ctx.Query().Get("status")

		orders, total, err := order.ListOrders(ctx, model.OrderFilter{
			Status:  status,
			Page:    page,
			PerPage: perPage,
		})
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "ORDER_LIST_FAILED", "Gagal mengambil daftar order.")
		}

		items := make([]orderListItemResponse, 0, len(orders))
		for _, item := range orders {
			items = append(items, newOrderListItemResponse(item))
		}
		return response.JSONPage(ctx, http.StatusOK, items, response.NewPagination(page, perPage, total))
	}
}

func cancelOrderHandler(order *usecase.OrderUseCase) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		result, err := order.CancelOrder(ctx, model.CancelOrderCommand{
			OrderID:       ctx.Vars().Get("id"),
			CorrelationID: correlationID(ctx),
			CausationID:   requestID(ctx),
			Reason:        "cancel_requested",
		})
		if errors.Is(err, model.ErrInvalidInput) {
			return response.JSONError(ctx, http.StatusBadRequest, "INVALID_ORDER_ID", "ID order tidak valid.")
		}
		if errors.Is(err, model.ErrOrderNotFound) {
			return response.JSONError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND", "Order tidak ditemukan.")
		}
		if errors.Is(err, model.ErrOrderConflict) {
			return response.JSONError(ctx, http.StatusConflict, "ORDER_CANNOT_BE_CANCELLED", "Order tidak bisa dibatalkan dari status saat ini.")
		}
		if err != nil {
			return response.JSONError(ctx, http.StatusInternalServerError, "ORDER_CANCEL_FAILED", "Gagal membatalkan order.")
		}
		return response.JSON(ctx, http.StatusOK, newOrderResponse(result))
	}
}

func orderItemInputs(items []createOrderItemRequest) []model.OrderItemInput {
	result := make([]model.OrderItemInput, 0, len(items))
	for _, item := range items {
		result = append(result, model.OrderItemInput{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}
	return result
}

func newOrderResponse(order model.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, orderItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}
	return orderResponse{
		ID:     order.ID,
		Status: order.Status,
		Customer: customerResponse{
			Name:    order.CustomerName,
			Phone:   order.CustomerPhone,
			Address: order.CustomerAddress,
		},
		Items:         items,
		TotalAmount:   order.TotalAmount,
		PaymentID:     order.PaymentID,
		CorrelationID: order.CorrelationID,
		CreatedAt:     formatTime(order.CreatedAt),
		UpdatedAt:     formatTime(order.UpdatedAt),
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func newOrderListItemResponse(order model.Order) orderListItemResponse {
	return orderListItemResponse{
		ID:     order.ID,
		Status: order.Status,
		Customer: customerResponse{
			Name:    order.CustomerName,
			Phone:   order.CustomerPhone,
			Address: order.CustomerAddress,
		},
		TotalAmount:   order.TotalAmount,
		PaymentID:     order.PaymentID,
		CorrelationID: order.CorrelationID,
		CreatedAt:     formatTime(order.CreatedAt),
		UpdatedAt:     formatTime(order.UpdatedAt),
	}
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
