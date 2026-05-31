package bootstrap

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	inventoryv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/inventory/v1"
	paymentv1 "github.com/odealidj/go-distributed-toko-bangunan/proto/payment/v1"
	orderkafka "github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/inbound/kafka"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/inbound/rest"
	outboundgrpc "github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/grpc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/adapter/outbound/postgres"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/saga"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/usecase"
	"github.com/odealidj/go-distributed-toko-bangunan/services/order-service/internal/application/worker"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/httpx"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/metrics"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/observability"
)

func NewApp(cfg config.ServiceConfig) (*kratos.App, func(), error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, err
	}

	inventoryConn, err := kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(cfg.InventoryGRPCAddr),
		kgrpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	paymentConn, err := kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(cfg.PaymentGRPCAddr),
		kgrpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		_ = inventoryConn.Close()
		pool.Close()
		return nil, nil, err
	}

	repository := postgres.NewOrderRepository(pool)
	outboxRepository := postgres.NewOutboxRepository(pool)
	inventoryClient := outboundgrpc.NewInventoryClient(inventoryv1.NewInventoryServiceClient(inventoryConn))
	paymentClient := outboundgrpc.NewPaymentClient(paymentv1.NewPaymentServiceClient(paymentConn))
	checkout := saga.NewCheckoutOrchestrator(repository, inventoryClient, paymentClient)
	order := usecase.NewOrder(checkout)
	paymentEventsConsumer, err := orderkafka.NewPaymentEventsConsumer(cfg, order)
	if err != nil {
		_ = paymentConn.Close()
		_ = inventoryConn.Close()
		pool.Close()
		return nil, nil, err
	}
	producer, err := messaging.NewKgoProducer(cfg.KafkaBrokers)
	if err != nil {
		paymentEventsConsumer.Close()
		_ = paymentConn.Close()
		_ = inventoryConn.Close()
		pool.Close()
		return nil, nil, err
	}
	outboxPublisher := worker.NewOutboxPublisher(outboxRepository, producer)
	workerCtx, stopWorker := context.WithCancel(context.Background())

	httpServer := khttp.NewServer(
		khttp.Address(cfg.HTTPAddr),
		khttp.Middleware(
			recovery.Recovery(),
			observability.ServerMetadata(),
			tracing.Server(),
		),
		khttp.Filter(httpx.Correlation()),
	)
	metrics.Register(httpServer)
	rest.RegisterRoutes(httpServer, cfg, order)

	app := kratos.New(
		kratos.Name(cfg.ServiceName),
		kratos.Server(httpServer),
		kratos.Logger(log.DefaultLogger),
		kratos.AfterStart(func(context.Context) error {
			go func() {
				if err := paymentEventsConsumer.Run(workerCtx); err != nil {
					log.Errorf("payment events consumer berhenti: %v", err)
				}
			}()
			go outboxPublisher.Run(workerCtx)
			return nil
		}),
		kratos.BeforeStop(func(context.Context) error {
			stopWorker()
			paymentEventsConsumer.Close()
			producer.Close()
			return nil
		}),
	)
	cleanup := func() {
		stopWorker()
		paymentEventsConsumer.Close()
		producer.Close()
		_ = paymentConn.Close()
		_ = inventoryConn.Close()
		pool.Close()
	}
	return app, cleanup, nil
}
