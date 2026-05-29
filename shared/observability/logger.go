package observability

import (
	"context"
	"log/slog"
	"os"
)

type contextHandler struct {
	next    slog.Handler
	service string
}

func NewLogger(service string) *slog.Logger {
	return slog.New(&contextHandler{
		next:    slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		service: service,
	})
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	nextRecord := slog.Record{
		Time:    record.Time,
		Message: record.Message,
		Level:   record.Level,
		PC:      record.PC,
	}
	record.Attrs(func(attr slog.Attr) bool {
		nextRecord.AddAttrs(attr)
		return true
	})

	nextRecord.AddAttrs(slog.String("service", h.service))
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		nextRecord.AddAttrs(slog.String("request_id", requestID))
	}
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		nextRecord.AddAttrs(slog.String("correlation_id", correlationID))
	}
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		nextRecord.AddAttrs(slog.String("trace_id", traceID))
	}
	if spanID := SpanIDFromContext(ctx); spanID != "" {
		nextRecord.AddAttrs(slog.String("span_id", spanID))
	}
	return h.next.Handle(ctx, nextRecord)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{
		next:    h.next.WithAttrs(attrs),
		service: h.service,
	}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{
		next:    h.next.WithGroup(name),
		service: h.service,
	}
}
