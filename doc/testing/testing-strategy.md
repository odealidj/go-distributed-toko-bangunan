# Strategi Testing

## 1. Tujuan Testing

- Verifikasi business rule.
- Verifikasi Saga success dan compensation flow.
- Verifikasi idempotency untuk duplicate command/event.
- Verifikasi service contract untuk REST, gRPC, dan Kafka.
- Verifikasi local demo scenario dapat diulang.
- Verifikasi Redis cache behavior tidak mengubah durable business correctness.
- Verifikasi REST response mengikuti standard envelope dan pagination contract.
- Verifikasi OpenTelemetry context propagation melewati REST, gRPC, dan Kafka.
- Verifikasi Kafka idempotent consumer behavior menggunakan inbox table.

## 2. Unit Tests

Unit test tidak boleh membutuhkan:

- PostgreSQL;
- Redis;
- Kafka;
- gRPC server;
- HTTP server;
- Docker Compose.

Unit test harus cepat, deterministik, dan dapat dijalankan dengan:

```text
go test ./...
```

Untuk application layer, gunakan fake port.

Contoh:

```text
FakeOrderRepository
FakeInventoryClient
FakePaymentClient
FakeOutboxRepository
FakeTransactionRunner
```

### order-service

- Perhitungan total order.
- Order status transition.
- Rule Saga step transition.
- Payment event handling.
- Duplicate event handling.
- Order cache invalidation setelah status transition.
- Payment event handler commit Kafka offset hanya setelah local transaction sukses.
- `CreateOrderUseCase` menyimpan product snapshot.
- `CheckoutSagaOrchestrator` memanggil reserve stock sebelum create payment.
- `HandlePaymentFailedUseCase` mempublish `OrderCancelled`.
- `HandlePaymentSucceededUseCase` mempublish `OrderConfirmed`.

### catalog-inventory-service

- Perhitungan available stock.
- Reserve stock success.
- Reserve stock failure ketika salah satu item insufficient.
- Release stock idempotency.
- Commit stock idempotency.
- Product cache hit/miss behavior.
- Product cache invalidation setelah product atau stock display berubah.
- Duplicate `OrderConfirmed` event tidak melakukan double commit stock.
- `ReserveStockUseCase` gagal secara atomic jika salah satu item insufficient.
- `ReleaseStockUseCase` aman dipanggil lebih dari sekali.
- `CommitStockUseCase` aman dipanggil lebih dari sekali.

### payment-service

- Create payment idempotency.
- Forced success mode.
- Forced failure mode.
- Cancel pending payment.
- Payment cache invalidation setelah status transition.
- Duplicate `OrderCancelled` event tidak membatalkan payment dua kali.
- `CreatePaymentUseCase` mempublish `PaymentCreated`.
- `CreatePaymentUseCase` mode `SUCCESS` mempublish `PaymentSucceeded`.
- `CreatePaymentUseCase` mode `FAILURE` mempublish `PaymentFailed`.

## 3. Integration Tests

Integration test memverifikasi adapter dan dependency nyata.

Dependency yang boleh digunakan:

- PostgreSQL;
- Redis;
- Kafka;
- gRPC server/client;
- HTTP server;
- OpenTelemetry Collector jika test observability.

Integration test boleh dijalankan dengan Docker Compose.

Command target:

```text
make test-integration
```

Aturan:

- setiap test harus menyiapkan data sendiri;
- setiap test harus dapat diulang;
- database harus direset atau memakai isolated schema/test database;
- Kafka topic test boleh memakai suffix unik;
- test tidak boleh bergantung pada urutan test lain.

### order-service integration

- Repository order dapat insert/find/update status.
- Outbox event tersimpan dalam transaction yang sama dengan order mutation.
- Inbox duplicate event ditolak oleh unique constraint.
- gRPC client ke inventory dapat timeout dengan benar.
- Kafka consumer memproses `PaymentSucceeded` dan mengupdate order.
- Kafka consumer memproses `PaymentFailed` dan mempublish `OrderCancelled`.

### catalog-inventory-service integration

- Repository product dapat read product dan stock.
- `ReserveStock` menggunakan transaction dan row lock yang benar.
- Concurrent reserve stock tidak membuat stock negatif.
- Consumer `OrderConfirmed` melakukan commit stock sekali.
- Consumer `OrderCancelled` melakukan release stock sekali.
- Product cache Redis miss membaca PostgreSQL lalu set cache.
- Redis down tidak merusak critical flow.

### payment-service integration

- Repository payment dapat create/update/find.
- `CreatePayment` idempotent berdasarkan `idempotency_key`.
- Payment mode `SUCCESS` mempublish `PaymentSucceeded`.
- Payment mode `FAILURE` mempublish `PaymentFailed`.
- Consumer `OrderCancelled` membatalkan pending payment secara idempotent.

### Kafka integration

- Producer mengirim header wajib.
- Consumer membaca header `traceparent` dan `x-correlation-id`.
- Manual offset commit hanya terjadi setelah local transaction sukses.
- Duplicate event dengan `event_id` sama tidak memicu business mutation kedua.
- Poison event masuk DLQ setelah retry limit.

### gRPC integration

- Server gRPC menerima metadata:
  - `traceparent`
  - `x-correlation-id`
  - `x-idempotency-key`
- Client gRPC menerapkan timeout.
- Error domain dimapping ke gRPC status yang sesuai.

### Redis integration

- Cache hit menghindari query DB untuk read non-critical.
- Cache miss fallback ke DB.
- Cache invalidation terjadi setelah status/data berubah.
- Redis unavailable diperlakukan sebagai cache miss untuk non-critical read.

## 4. Component Tests

Component test menjalankan satu service dengan dependency nyata, tetapi downstream service diganti fake server.

Contoh:

```text
order-service + PostgreSQL + Redis + Kafka + fake inventory gRPC + fake payment gRPC
```

Tujuan:

- menguji satu service secara lebih realistis;
- tetap lebih stabil daripada E2E penuh;
- bagus untuk menguji adapter dan use case bersama.

Skenario component test `order-service`:

- checkout success dengan fake inventory dan fake payment;
- stock reservation gagal dari fake inventory;
- payment gagal dari fake payment;
- timeout inventory menghasilkan status yang tepat;
- duplicate Kafka event aman.

## 5. Contract Tests

- REST API memvalidasi `doc/api/openapi.yaml`.
- Contoh REST response sesuai dengan `doc/api/response-standard.md`.
- gRPC client dan server compile dari file di `doc/grpc`.
- Kafka event valid terhadap `doc/events/asyncapi.yaml` atau JSON schema.
- Kafka headers menyertakan `traceparent`, `x-correlation-id`, `x-causation-id`, `x-event-id`, dan `x-event-type`.

## 6. Architecture Tests

- Domain package tidak import adapter package.
- Domain package tidak import SQL, Redis, Kafka, HTTP framework, atau generated gRPC client package.
- Application package bergantung pada port/interface, bukan concrete adapter.
- Inbound adapter mendelegasikan ke application use case.
- Domain package tidak import `github.com/go-kratos/kratos`.
- Domain/application package tidak import `pgx`, `sqlc` generated package, `sqlx`, `kgo`, atau `go-redis`.
- `sqlc` hanya muncul di outbound postgres adapter milik `order-service` dan `catalog-inventory-service`.
- `sqlx` hanya muncul di outbound postgres adapter milik `payment-service`.

## 7. Observability Tests

- Incoming REST request dengan `traceparent` mempertahankan trace yang sama melalui gRPC call.
- Kafka producer menulis trace context headers.
- Kafka consumer melanjutkan trace context dari Kafka headers.
- Log menyertakan `trace_id`, `span_id`, `correlation_id`, dan `order_id` jika berlaku.
- Duplicate Kafka event menaikkan duplicate metric dan tidak melakukan business state mutation dua kali.

## 8. End-to-End Demo Tests

### Scenario 1: Checkout Success

Kondisi:

- Product stock tersedia.
- `payment_mode` adalah `SUCCESS`.

Hasil yang diharapkan:

- Order status menjadi `CONFIRMED`.
- Payment status menjadi `SUCCEEDED`.
- Stock reservation menjadi `COMMITTED`.
- Inventory `on_hand_qty` berkurang tepat satu kali.

### Scenario 2: Insufficient Stock

Kondisi:

- Requested quantity lebih besar dari available stock.

Hasil yang diharapkan:

- Order status menjadi `REJECTED`.
- Tidak ada payment yang dibuat.
- Stock tidak di-reserve.

### Scenario 3: Payment Failed

Kondisi:

- Product stock tersedia.
- `payment_mode` adalah `FAILURE`.

Hasil yang diharapkan:

- Order status menjadi `CANCELLED`.
- Payment status menjadi `FAILED`.
- Stock reservation menjadi `RELEASED`.
- Available stock kembali ke nilai awal.

### Scenario 4: Duplicate PaymentSucceeded Event

Kondisi:

- Order sudah confirmed.
- Event `PaymentSucceeded` yang sama dikirim lagi.

Hasil yang diharapkan:

- Order tetap `CONFIRMED`.
- `OrderConfirmed` tidak memiliki duplicate business effect.
- Stock di-commit hanya sekali.

### Scenario 5: Duplicate OrderCancelled Event

Kondisi:

- Stock reservation sudah released.
- Event `OrderCancelled` yang sama dikirim lagi.

Hasil yang diharapkan:

- Reservation tetap `RELEASED`.
- Available stock tidak bertambah dua kali.

### Scenario 6: Redis Unavailable

Kondisi:

- Redis dihentikan.

Hasil yang diharapkan:

- Product read fallback ke PostgreSQL.
- Order read fallback ke PostgreSQL.
- Checkout success/failure behavior tetap benar.
- Durable idempotency tetap berjalan melalui database table.

## 9. Test Pyramid

Target proporsi:

```text
Unit test        : paling banyak
Integration test : sedang
Component test   : sedang
E2E test          : sedikit tapi kritikal
Manual demo       : untuk presentasi
```

Prioritas implementasi test:

1. Domain state machine unit test.
2. Use case unit test dengan fake port.
3. Repository integration test.
4. Kafka consumer idempotency integration test.
5. Saga component test.
6. E2E checkout success/failure.

## 10. Naming Convention

Gunakan nama test yang menjelaskan kondisi dan ekspektasi.

Contoh:

```text
TestReserveStock_WhenStockInsufficient_ReturnsErrorAndDoesNotReserveAnyItem
TestHandlePaymentFailed_WhenOrderWaitingPayment_CancelsOrderAndPublishesOrderCancelled
TestCommitStock_WhenDuplicateOrderConfirmed_DoesNotReduceStockTwice
```

## 11. CI Test Split

CI minimal:

```text
make test-unit
make test-contract
make build-all
```

CI lengkap:

```text
make test-unit
make test-integration
make test-component
make test-e2e
```

Untuk awal, integration/E2E boleh dijalankan manual jika CI resource terbatas.
