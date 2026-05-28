package response

import (
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type Meta struct {
	RequestID     string `json:"request_id"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

type Success struct {
	OK         bool        `json:"success"`
	Data       any         `json:"data"`
	Meta       Meta        `json:"meta"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type Error struct {
	OK    bool      `json:"success"`
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

func JSON(ctx khttp.Context, status int, data any) error {
	return ctx.JSON(status, Success{
		OK:   true,
		Data: data,
		Meta: meta(ctx),
	})
}

func JSONPage(ctx khttp.Context, status int, data any, pagination Pagination) error {
	return ctx.JSON(status, Success{
		OK:         true,
		Data:       data,
		Meta:       meta(ctx),
		Pagination: &pagination,
	})
}

func NewPagination(page, perPage, totalItems int) Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if totalItems < 0 {
		totalItems = 0
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + perPage - 1) / perPage
	}

	return Pagination{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    totalPages > 0 && page < totalPages,
		HasPrev:    page > 1 && totalPages > 0,
	}
}

func JSONError(ctx khttp.Context, status int, code, message string) error {
	return ctx.JSON(status, Error{
		OK: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: meta(ctx),
	})
}

func meta(ctx khttp.Context) Meta {
	req := ctx.Request()
	requestID := req.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = req.Header.Get("X-Request-ID")
	}
	correlationID := req.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		correlationID = req.Header.Get("X-Correlation-ID")
	}
	if requestID == "" {
		requestID = "local-request"
	}
	if correlationID == "" {
		correlationID = requestID
	}
	return Meta{
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
}

func NotFound(ctx khttp.Context) error {
	return JSONError(ctx, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Resource tidak ditemukan.")
}
