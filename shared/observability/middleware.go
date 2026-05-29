package observability

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

func ServerMetadata() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return next(ctx, req)
			}

			headers := tr.RequestHeader()
			requestID := firstValue(headers.Get(HeaderRequestID), headers.Get(HeaderRequestIDAlt))
			if requestID == "" {
				requestID = NewExecutionID("req")
				headers.Set(HeaderRequestID, requestID)
			}

			correlationID := firstValue(headers.Get(HeaderCorrelationID), headers.Get(HeaderCorrelationAlt))
			if correlationID == "" {
				correlationID = requestID
				headers.Set(HeaderCorrelationID, correlationID)
			}

			if reply := tr.ReplyHeader(); reply != nil {
				reply.Set(HeaderRequestID, requestID)
				reply.Set(HeaderCorrelationID, correlationID)
			}

			return next(WithRequestScope(ctx, requestID, correlationID), req)
		}
	}
}

func firstValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
