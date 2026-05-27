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

