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
	OK   bool `json:"success"`
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
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

func JSON(ctx khttp.Context, status int, data any) error {
	return ctx.JSON(status, Success{
		OK:   true,
		Data: data,
		Meta: meta(ctx),
	})
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
