package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/bootstrap"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/observability"
)

func main() {
	cfg := config.LoadService("catalog-inventory-service", ":8081", ":9001")
	slog.SetDefault(observability.NewLogger(cfg.ServiceName))

	shutdownObservability, err := observability.Init(context.Background(), cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		slog.Error("gagal menyiapkan observability", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownObservability(context.Background()); err != nil {
			slog.Error("gagal shutdown observability", "error", err)
		}
	}()

	app, cleanup, err := bootstrap.NewApp(cfg)
	if err != nil {
		slog.Error("gagal menyiapkan service", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		slog.Error("service berhenti dengan error", "error", err)
		os.Exit(1)
	}
}
