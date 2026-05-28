package bootstrap

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	inventoryv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/inventory/v1"
	inventorygrpc "github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/adapter/inbound/grpc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/adapter/inbound/rest"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/adapter/outbound/postgres"
	redisadapter "github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/adapter/outbound/redis"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/httpx"
	goredis "github.com/redis/go-redis/v9"
)

func NewApp(cfg config.ServiceConfig) (*kratos.App, func(), error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL wajib diisi")
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, err
	}

	redisClient := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})
	repository := postgres.NewCatalogRepository(pool)
	cache := redisadapter.NewProductCache(redisClient)
	catalog := usecase.NewCatalog(repository, cache)

	httpServer := khttp.NewServer(
		khttp.Address(cfg.HTTPAddr),
		khttp.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
		khttp.Filter(httpx.Correlation()),
	)
	rest.RegisterRoutes(httpServer, cfg, catalog)

	grpcServer := kgrpc.NewServer(
		kgrpc.Address(cfg.GRPCAddr),
		kgrpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
	)
	inventoryv1.RegisterInventoryServiceServer(grpcServer, inventorygrpc.NewInventoryServer(catalog))

	app := kratos.New(
		kratos.Name(cfg.ServiceName),
		kratos.Server(httpServer, grpcServer),
		kratos.Logger(log.DefaultLogger),
	)
	cleanup := func() {
		redisClient.Close()
		pool.Close()
	}
	return app, cleanup, nil
}
