module github.com/odealidj/go-distributed-toko-bangunan/services/order-service

go 1.22

require (
	github.com/go-kratos/kratos/v2 v2.9.2
	github.com/jackc/pgx/v5 v5.6.0
	github.com/odealidj/go-distributed-toko-bangunan/proto v0.0.0
	github.com/odealidj/go-distributed-toko-bangunan/shared v0.0.0
	google.golang.org/grpc v1.64.1
)

require (
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/odealidj/go-distributed-toko-bangunan/shared => ../../shared

replace github.com/odealidj/go-distributed-toko-bangunan/proto => ../../proto
