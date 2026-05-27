# Panduan Go Hexagonal Architecture

## 1. Keputusan

Semua service diimplementasikan dengan Go menggunakan Hexagonal Architecture.

Tujuannya adalah menjaga domain dan application logic tetap independen dari detail transport, storage, cache, dan messaging.

## 2. Aturan Layering

```text
Inbound adapters  -> Application layer -> Domain layer
                                  |
                                  v
                         Outbound ports
                                  |
                                  v
                         Outbound adapters
```

Aturan:

- Domain layer tidak boleh import HTTP, gRPC, Kafka, Redis, SQL, atau framework package.
- Application layer mengoordinasikan use case, transaction, port, dan domain state transition.
- Port adalah Go interface yang dimiliki sisi application/domain.
- Adapter mengimplementasikan port menggunakan teknologi konkret.
- REST, gRPC, dan Kafka consumer adalah inbound adapter.
- PostgreSQL, Redis, Kafka producer, dan gRPC client adalah outbound adapter.

## 3. Library Go yang Disarankan

Pilihan library boleh berubah saat implementasi, tetapi berikut default yang wajar:

| Concern | Suggested Option |
| --- | --- |
| Framework | `go-kratos` |
| HTTP/gRPC transport | `go-kratos` di adapter/bootstrap layer |
| PostgreSQL order-service | `sqlc` + `pgx` / `pgxpool` |
| PostgreSQL catalog-inventory-service | `sqlc` + `pgx` / `pgxpool` |
| PostgreSQL payment-service | `sqlx` + `pgx` stdlib driver |
| SQL migrations | `goose` |
| Redis | `go-redis` |
| Kafka | `github.com/twmb/franz-go/pkg/kgo` |
| Logging | standard `log/slog` |
| Configuration | environment variables dengan typed config struct |
| Testing | standard `testing`, fake port, dan Docker Compose untuk integration test |

Jangan biarkan API library bocor ke domain layer.

Keputusan teknologi final dijelaskan di:

```text
doc/implementation/technology-decisions.md
```

## 4. Layout Service yang Direkomendasikan

Setiap service sebaiknya mengikuti struktur berikut:

```text
{service-name}/
  cmd/
    api/
      main.go
    worker/
      main.go
  internal/
    domain/
      model/
      event/
      valueobject/
      service/
    application/
      command/
      query/
      port/
      usecase/
      saga/              # order-service only
    adapter/
      inbound/
        rest/
        grpc/
        kafka/
      outbound/
        postgres/
          query/          # service sqlc only
          sqlc/           # service sqlc only
        redis/
        kafka/
        grpc/
        payment_gateway/
    config/
    bootstrap/
  migrations/
  test/
```

Untuk service kecil, `cmd/api` dan `cmd/worker` boleh digabung dalam satu process, tetapi inbound adapter harus tetap dipisahkan di kode.

## 5. Tanggung Jawab Package

### domain

Berisi business concept murni:

- entities
- value objects
- domain events
- state transition rules
- domain services

Examples:

- `Order`
- `OrderItem`
- `Payment`
- `StockReservation`
- `OrderStatus`
- `PaymentStatus`

### application

Berisi use case dan orchestration:

- `CreateOrder`
- `CancelOrder`
- `ReserveStock`
- `CreatePayment`
- `HandlePaymentSucceeded`
- `HandlePaymentFailed`
- `CheckoutSagaOrchestrator`

Application layer boleh bergantung pada port, tetapi tidak pada concrete adapter.

### adapter/inbound/rest

Menangani HTTP yang menghadap frontend:

- request validation
- response mapping
- correlation ID extraction
- pemanggilan application use case

Business workflow tidak boleh berada di REST handler.

### adapter/inbound/grpc

Menangani internal gRPC server method:

- request mapping
- metadata extraction
- pemanggilan application use case

### adapter/inbound/kafka

Menangani Kafka event consumption:

- deserialize event envelope
- cek inbox/idempotency melalui application port
- panggil event handling use case
- commit offset hanya setelah processing sukses

### adapter/outbound/postgres

Mengimplementasikan repository dan transactional store:

- repositories
- outbox store
- inbox store
- unit of work / transaction runner

### adapter/outbound/redis

Mengimplementasikan cache port:

- product detail cache
- product list cache
- order read model cache when useful
- short-lived idempotency locks when needed

Redis tidak boleh menjadi source of truth.

### adapter/outbound/kafka

Mengimplementasikan event publisher:

- publish outbox event ke Kafka
- gunakan `order_id` sebagai message key untuk event terkait checkout
- pertahankan event envelope

### adapter/outbound/grpc

Mengimplementasikan internal service client:

- inventory client yang digunakan order-service
- payment client yang digunakan order-service

## 6. Contoh Go Interface

Outbound port yang dimiliki application:

```go
type InventoryClient interface {
    ValidateProducts(ctx context.Context, items []OrderItemInput) ([]ValidatedOrderItem, error)
    ReserveStock(ctx context.Context, command ReserveStockCommand) (*StockReservationResult, error)
}
```

Repository port:

```go
type OrderRepository interface {
    Save(ctx context.Context, tx Tx, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
}
```

Cache port:

```go
type ProductCache interface {
    GetProduct(ctx context.Context, productID string) (*ProductSnapshot, error)
    SetProduct(ctx context.Context, product ProductSnapshot, ttl time.Duration) error
    DeleteProduct(ctx context.Context, productID string) error
}
```

## 7. Arah Dependency

Diizinkan:

```text
adapter -> application -> domain
adapter -> application port
adapter -> domain DTO/value object when needed
```

Dilarang:

```text
domain -> adapter
domain -> database
domain -> Redis
domain -> Kafka
domain -> gRPC generated client
application -> concrete PostgreSQL repository
application -> concrete Kafka producer
```

## 8. Penempatan Saga di Order Service

Checkout Saga orchestrator berada di:

```text
order-service/internal/application/saga
```

Module ini boleh bergantung pada port berikut:

- `OrderRepository`
- `SagaRepository`
- `OutboxRepository`
- `InventoryClient`
- `PaymentClient`
- `TransactionRunner`

Module ini tidak boleh bergantung langsung pada:

- generated gRPC client
- Kafka producer implementation
- SQL driver
- sqlc generated code
- sqlx DB/Tx
- Redis client

## 9. Process Model

Runtime process yang direkomendasikan per service:

```text
api process:
  - REST server
  - gRPC server when service exposes gRPC

worker process:
  - Kafka consumers
  - outbox publisher
  - scheduled cleanup/repair jobs
```

Untuk kesederhanaan demo, satu binary boleh menjalankan API dan worker, tetapi kode tetap harus memisahkan adapter.

## 10. Error Handling

- Domain error harus typed.
- Application layer memetakan domain error ke use case error.
- REST adapter memetakan use case error ke HTTP status.
- gRPC adapter memetakan use case error ke gRPC status code.
- Kafka adapter memperlakukan retryable dan non-retryable error secara berbeda.

Contoh domain error:

- `ErrInsufficientStock`
- `ErrInvalidOrderStatus`
- `ErrDuplicateEvent`
- `ErrPaymentAlreadyFinalized`

## 11. Pendekatan Testing

- Domain test tidak boleh membutuhkan database, Redis, Kafka, atau gRPC.
- Application test harus menggunakan fake port.
- Adapter test boleh menggunakan PostgreSQL/Redis/Kafka nyata melalui test container atau local Docker.
- Contract test harus memvalidasi compatibility OpenAPI, proto, dan event schema.

Data access rule:

- `order-service` dan `catalog-inventory-service` menggunakan `sqlc` di outbound postgres adapter.
- `payment-service` menggunakan `sqlx` di outbound postgres adapter.
- Domain/application tidak boleh import `sqlc`, `sqlx`, `pgx`, atau `database/sql`.
