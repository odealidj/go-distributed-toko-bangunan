package main

import (
	"log/slog"
	"os"

	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/bootstrap"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
)

func main() {
	cfg := config.LoadService("catalog-inventory-service", ":8081", ":9001")
	app := bootstrap.NewApp(cfg)

	if err := app.Run(); err != nil {
		slog.Error("service berhenti dengan error", "error", err)
		os.Exit(1)
	}
}
