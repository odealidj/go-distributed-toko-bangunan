package bootstrap

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/odealidj/go-distributed-toko-bangunan/services/payment-service/internal/adapter/inbound/rest"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/config"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/httpx"
)

func NewApp(cfg config.ServiceConfig) *kratos.App {
	httpServer := khttp.NewServer(
		khttp.Address(cfg.HTTPAddr),
		khttp.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
		khttp.Filter(httpx.Correlation()),
	)
	rest.RegisterRoutes(httpServer, cfg)

	return kratos.New(
		kratos.Name(cfg.ServiceName),
		kratos.Server(httpServer),
		kratos.Logger(log.DefaultLogger),
	)
}
