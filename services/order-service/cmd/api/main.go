package main

import (
	"log/slog"
	"os"

	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/bootstrap"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
)

func main() {
	cfg := config.LoadService("order-service", ":8080", ":9000")
	app, cleanup, err := bootstrap.NewApp(cfg)
	if err != nil {
		slog.Error("gagal membuat aplikasi", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		slog.Error("service berhenti dengan error", "error", err)
		os.Exit(1)
	}
}
