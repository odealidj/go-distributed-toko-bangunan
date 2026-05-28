package observability

import "log/slog"

func NewLogger(service string) *slog.Logger {
	return slog.Default().With("service", service)
}
