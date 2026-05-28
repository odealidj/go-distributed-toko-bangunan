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
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	paymentv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/payment/v1"
	paymentgrpc "github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/adapter/inbound/grpc"
	paymentkafka "github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/adapter/inbound/kafka"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/adapter/inbound/rest"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/adapter/outbound/postgres"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/httpx"
)

func NewApp(cfg config.ServiceConfig) (*kratos.App, func(), error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL wajib diisi")
	}

	db, err := sqlx.ConnectContext(context.Background(), "pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	repository := postgres.NewPaymentRepository(db)
	payment := usecase.NewPayment(repository)
	orderEventsConsumer, err := paymentkafka.NewOrderEventsConsumer(cfg, payment)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	consumerCtx, stopConsumer := context.WithCancel(context.Background())

	httpServer := khttp.NewServer(
		khttp.Address(cfg.HTTPAddr),
		khttp.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
		khttp.Filter(httpx.Correlation()),
	)
	rest.RegisterRoutes(httpServer, cfg, payment)

	grpcServer := kgrpc.NewServer(
		kgrpc.Address(cfg.GRPCAddr),
		kgrpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
	)
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentgrpc.NewPaymentServer(payment))

	app := kratos.New(
		kratos.Name(cfg.ServiceName),
		kratos.Server(httpServer, grpcServer),
		kratos.Logger(log.DefaultLogger),
		kratos.AfterStart(func(context.Context) error {
			go func() {
				if err := orderEventsConsumer.Run(consumerCtx); err != nil {
					log.Errorf("order events consumer berhenti: %v", err)
				}
			}()
			return nil
		}),
		kratos.BeforeStop(func(context.Context) error {
			stopConsumer()
			orderEventsConsumer.Close()
			return nil
		}),
	)
	cleanup := func() {
		stopConsumer()
		orderEventsConsumer.Close()
		_ = db.Close()
	}
	return app, cleanup, nil
}
