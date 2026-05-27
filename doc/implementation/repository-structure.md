# Struktur Repository

## 1. Tujuan

Dokumen ini mendefinisikan struktur repository untuk implementasi monorepo 3 service dengan Go, go-kratos, dan Hexagonal Architecture.

## 2. Struktur Root

```text
.
├── go.work
├── Makefile
├── docker-compose.yml
├── README.md
├── doc/
├── proto/
│   ├── inventory/v1/
│   ├── order/v1/
│   └── payment/v1/
├── services/
│   ├── order-service/
│   ├── catalog-inventory-service/
│   └── payment-service/
├── shared/
│   ├── go.mod
│   ├── observability/
│   ├── response/
│   ├── messaging/
│   └── config/
├── scripts/
├── deployments/
│   ├── docker/
│   └── otel/
└── tests/
    ├── e2e/
    └── integration/
```

## 3. Aturan Root Folder

| Folder/File | Tanggung jawab |
| --- | --- |
| `go.work` | Menghubungkan semua Go module service dan shared module. |
| `Makefile` | Command operasional lokal. |
| `docker-compose.yml` | Infrastruktur lokal dan service runtime. |
| `doc/` | Dokumentasi arsitektur dan contract. |
| `proto/` | Source `.proto` dan generated code jika dipilih. |
| `services/` | Semua service utama. |
| `shared/` | Utility bersama yang tidak membawa business logic. |
| `scripts/` | Script demo/test/migration. |
| `deployments/` | Config deployment lokal. |
| `tests/` | E2E dan integration test lintas service. |

## 3.1 Go Module Strategy

Repository menggunakan `go.work` di root dan `go.mod` per service.

Struktur:

```text
go.work
services/order-service/go.mod
services/catalog-inventory-service/go.mod
services/payment-service/go.mod
shared/go.mod
```

Aturan:

- tidak ada root `go.mod`;
- setiap service memiliki dependency sendiri;
- `shared/go.mod` hanya berisi utility generic, bukan business logic;
- `go.work` digunakan untuk development lokal agar semua module mudah dibuild bersama;
- build Docker service tetap dilakukan dari folder service masing-masing.

Contoh isi `go.work`:

```text
go 1.22

use (
  ./shared
  ./services/order-service
  ./services/catalog-inventory-service
  ./services/payment-service
)
```

Alasan:

- menjaga isolasi dependency per service;
- tetap nyaman untuk monorepo;
- selaras dengan microservices boundary.

## 4. Struktur Per Service

```text
services/{service-name}/
├── go.mod
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   ├── event/
│   │   ├── valueobject/
│   │   └── service/
│   ├── application/
│   │   ├── command/
│   │   ├── query/
│   │   ├── port/
│   │   ├── usecase/
│   │   └── saga/
│   ├── adapter/
│   │   ├── inbound/
│   │   │   ├── rest/
│   │   │   ├── grpc/
│   │   │   └── kafka/
│   │   └── outbound/
│   │       ├── postgres/
│   │       ├── redis/
│   │       ├── kafka/
│   │       ├── grpc/
│   │       └── payment_gateway/
│   ├── config/
│   └── bootstrap/
├── migrations/
└── test/
```

Catatan:

- Folder `saga/` hanya wajib di `order-service`.
- Folder `payment_gateway/` terutama relevan untuk `payment-service`.
- Folder boleh disederhanakan jika service belum membutuhkan semua adapter, tetapi arah dependency tetap sama.

## 4.1 Struktur Data Access Per Service

### order-service

Gunakan `sqlc` + `pgx`.

```text
services/order-service/internal/adapter/outbound/postgres/
├── query/
│   ├── orders.sql
│   ├── order_items.sql
│   ├── saga.sql
│   ├── outbox.sql
│   └── inbox.sql
├── sqlc/
│   └── generated files
├── order_repository.go
├── saga_repository.go
├── outbox_repository.go
├── inbox_repository.go
└── transaction_runner.go
```

### catalog-inventory-service

Gunakan `sqlc` + `pgx`.

```text
services/catalog-inventory-service/internal/adapter/outbound/postgres/
├── query/
│   ├── products.sql
│   ├── inventories.sql
│   ├── stock_reservations.sql
│   ├── outbox.sql
│   └── inbox.sql
├── sqlc/
│   └── generated files
├── product_repository.go
├── inventory_repository.go
├── stock_reservation_repository.go
├── outbox_repository.go
├── inbox_repository.go
└── transaction_runner.go
```

### payment-service

Gunakan `sqlx` + `pgx` stdlib driver.

```text
services/payment-service/internal/adapter/outbound/postgres/
├── payment_repository_sqlx.go
├── payment_attempt_repository_sqlx.go
├── outbox_repository_sqlx.go
├── inbox_repository_sqlx.go
└── transaction_runner_sqlx.go
```

Alasan:

- `order-service` dan `catalog-inventory-service` lebih kritikal dan query-nya lebih sensitif, sehingga menggunakan `sqlc` untuk type-safety.
- `payment-service` lebih sederhana dan cocok untuk pembelajaran `sqlx`.
- Perbedaan library tidak boleh terlihat oleh application layer karena semua akses DB melewati port/interface.

## 5. Dependency Rule

Aturan dependency:

```text
adapter -> application -> domain
```

Dilarang:

```text
domain -> adapter
domain -> go-kratos
domain -> pgx
domain -> sqlc generated package
domain -> sqlx
domain -> Redis client
domain -> Kafka client
domain -> generated gRPC client
application -> concrete adapter
```

## 6. go-kratos Placement

`go-kratos` hanya digunakan di:

```text
internal/adapter/inbound/rest
internal/adapter/inbound/grpc
internal/bootstrap
```

Domain dan application tidak boleh import package Kratos.

## 7. Payment Gateway Pihak Ketiga

Jika `payment-service` perlu call API pihak ketiga, tambahkan outbound adapter:

```text
services/payment-service/internal/adapter/outbound/payment_gateway/
  mock/
  midtrans/
  xendit/
```

Application hanya tahu port:

```go
type PaymentGateway interface {
    Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
    Cancel(ctx context.Context, paymentID string) error
}
```

Adapter pihak ketiga bertanggung jawab untuk:

- call HTTP/API provider;
- mapping status provider ke domain status;
- retry provider call jika aman;
- masking secret di log;
- propagasi trace context jika memungkinkan.

Domain tidak boleh tahu status code atau response shape provider.

## 8. Shared Package Rule

`shared/` hanya boleh berisi generic utility seperti:

- response envelope;
- correlation ID helper;
- OpenTelemetry bootstrap helper;
- config loader;
- Kafka event envelope struct;
- logger helper.

`shared/` tidak boleh berisi business rule domain toko bangunan.

## 9. Test Placement

Unit test ditempatkan dekat package yang dites:

```text
internal/domain/model/order_test.go
internal/application/usecase/create_order_test.go
```

Integration test per service:

```text
services/{service-name}/test/integration/
```

E2E test lintas service:

```text
tests/e2e/
```

## 10. Makefile Target

Root `Makefile` menjadi entrypoint operasional lokal.

### Infra target

Target khusus infrastructure:

```text
make infra-up
make infra-down
make infra-logs
make infra-ps
```

`make infra-up` hanya menjalankan:

```text
postgres
redis
kafka
kafka-ui
otel-collector
jaeger
prometheus
grafana
exporter jika sudah diaktifkan
```

Service aplikasi tidak ikut dijalankan agar mudah debug dari IDE atau terminal lokal.

### Service run target

```text
make order-run
make inventory-run
make payment-run
```

### Service build target

```text
make order-build
make inventory-build
make payment-build
make build-all
```

### Service test target

```text
make order-test
make inventory-test
make payment-test
make test-all
```

### Migration target

```text
make order-migrate
make inventory-migrate
make payment-migrate
make migrate
```

### Generation target

```text
make proto
make sqlc
make generate
```

### Demo target

```text
make demo-success
make demo-payment-failed
make demo-insufficient-stock
make demo-duplicate-event
```

### Performance target

```text
make perf-smoke
make perf-load
make perf-stress
make perf-spike
```

## 11. Container Engine

Local development menggunakan Podman, tetapi deployment VPS boleh menggunakan Docker.

Makefile harus mendukung override:

```text
COMPOSE ?= podman compose
```

Di VPS:

```text
make infra-up COMPOSE="docker compose"
```

Aturan agar Compose portable:

- gunakan service DNS seperti `kafka:9092`, `postgres:5432`, `redis:6379`;
- jangan bergantung pada `host.docker.internal`;
- hindari `container_name` kecuali benar-benar perlu;
- gunakan named volume untuk data utama;
- hati-hati dengan bind mount permission pada Podman rootless;
- jika memakai SELinux, bind mount mungkin membutuhkan suffix `:Z`;
- jangan memakai fitur Compose yang hanya berjalan di Docker Swarm.
