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

### order-service

- Perhitungan total order.
- Order status transition.
- Rule Saga step transition.
- Payment event handling.
- Duplicate event handling.
- Order cache invalidation setelah status transition.
- Payment event handler commit Kafka offset hanya setelah local transaction sukses.

### catalog-inventory-service

- Perhitungan available stock.
- Reserve stock success.
- Reserve stock failure ketika salah satu item insufficient.
- Release stock idempotency.
- Commit stock idempotency.
- Product cache hit/miss behavior.
- Product cache invalidation setelah product atau stock display berubah.
- Duplicate `OrderConfirmed` event tidak melakukan double commit stock.

### payment-service

- Create payment idempotency.
- Forced success mode.
- Forced failure mode.
- Cancel pending payment.
- Payment cache invalidation setelah status transition.
- Duplicate `OrderCancelled` event tidak membatalkan payment dua kali.

## 3. Integration Tests

- `order-service` memanggil `ValidateProducts` dan menyimpan product snapshot.
- `order-service` memanggil `ReserveStock` sebelum `CreatePayment`.
- `catalog-inventory-service` mempublish `StockReserved` setelah reservation.
- `payment-service` mempublish `PaymentSucceeded` atau `PaymentFailed`.
- `order-service` mengonsumsi payment event dan mempublish final order event.

## 4. Contract Tests

- REST API memvalidasi `doc/api/openapi.yaml`.
- Contoh REST response sesuai dengan `doc/api/response-standard.md`.
- gRPC client dan server compile dari file di `doc/grpc`.
- Kafka event valid terhadap `doc/events/asyncapi.yaml` atau JSON schema.
- Kafka headers menyertakan `traceparent`, `x-correlation-id`, `x-causation-id`, `x-event-id`, dan `x-event-type`.

## 5. Architecture Tests

- Domain package tidak import adapter package.
- Domain package tidak import SQL, Redis, Kafka, HTTP framework, atau generated gRPC client package.
- Application package bergantung pada port/interface, bukan concrete adapter.
- Inbound adapter mendelegasikan ke application use case.

## 6. Observability Tests

- Incoming REST request dengan `traceparent` mempertahankan trace yang sama melalui gRPC call.
- Kafka producer menulis trace context headers.
- Kafka consumer melanjutkan trace context dari Kafka headers.
- Log menyertakan `trace_id`, `span_id`, `correlation_id`, dan `order_id` jika berlaku.
- Duplicate Kafka event menaikkan duplicate metric dan tidak melakukan business state mutation dua kali.

## 7. End-to-End Demo Tests

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
