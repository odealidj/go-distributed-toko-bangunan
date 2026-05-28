package main

import (
	"log/slog"
	"os"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/bootstrap"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
)

func main() {
	cfg := config.LoadService("payment-service", ":8082", ":9002")
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
