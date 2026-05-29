package metrics

import (
	"net/http"
	"sync"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	setupOnce sync.Once
	setupErr  error
)

func Setup(service string) error {
	setupOnce.Do(func() {
		serviceInfo := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "toko_service_info",
				Help: "Static service information for Mini Toko Bangunan.",
				ConstLabels: prometheus.Labels{
					"service": service,
				},
			},
			[]string{"version"},
		)
		serviceInfo.WithLabelValues("demo").Set(1)
		if err := prometheus.Register(serviceInfo); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				setupErr = err
			}
		}
	})
	return setupErr
}

func Register(server *khttp.Server) {
	server.Handle("/metrics", httpMetricsHandler())
}

func httpMetricsHandler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}
