package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

const (
	HeaderRequestID      = "X-Request-Id"
	HeaderRequestIDAlt   = "X-Request-ID"
	HeaderCorrelationID  = "X-Correlation-Id"
	HeaderCorrelationAlt = "X-Correlation-ID"
)

type requestScopeKey struct{}

type requestScope struct {
	RequestID     string
	CorrelationID string
}

func WithRequestScope(ctx context.Context, requestID, correlationID string) context.Context {
	if requestID == "" && correlationID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestScopeKey{}, requestScope{
		RequestID:     requestID,
		CorrelationID: correlationID,
	})
}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	scope := requestScopeFromContext(ctx)
	scope.CorrelationID = correlationID
	if scope.RequestID == "" && correlationID != "" {
		scope.RequestID = NewExecutionID("req")
	}
	return context.WithValue(ctx, requestScopeKey{}, scope)
}

func RequestIDFromContext(ctx context.Context) string {
	return requestScopeFromContext(ctx).RequestID
}

func CorrelationIDFromContext(ctx context.Context) string {
	return requestScopeFromContext(ctx).CorrelationID
}

func NewExecutionID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_local"
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func requestScopeFromContext(ctx context.Context) requestScope {
	scope, _ := ctx.Value(requestScopeKey{}).(requestScope)
	return scope
}
