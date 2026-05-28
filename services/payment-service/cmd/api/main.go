package main

import (
	"log/slog"
	"os"

	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/bootstrap"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
)

func main() {
	cfg := config.LoadService("payment-service", ":8082", ":9002")
	app := bootstrap.NewApp(cfg)

	if err := app.Run(); err != nil {
		slog.Error("service berhenti dengan error", "error", err)
		os.Exit(1)
	}
}
