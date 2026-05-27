# Panduan Implementasi AI

Dokumen ini memberikan batasan implementasi untuk AI agent atau engineer. Tujuannya adalah menjaga keputusan arsitektur saat menghasilkan implementasi Go.

## 1. Batasan Arsitektur

- Implementasikan service menggunakan Go.
- Gunakan Hexagonal Architecture untuk setiap service.
- Pertahankan boundary 3 service:
  - `order-service`
  - `catalog-inventory-service`
  - `payment-service`
- Jangan gabungkan database antar service.
- Jangan lakukan cross-service database join.
- Gunakan REST hanya untuk API yang menghadap frontend.
- Gunakan gRPC untuk command/query sinkron internal.
- Gunakan Kafka untuk domain event.
- Gunakan Redis sebagai cache dan optional short-lived lock provider.
- Gunakan PostgreSQL sebagai source of truth.
- Pertahankan checkout orchestration di dalam `order-service` untuk MVP.
- Implementasikan orchestration sebagai module/package terpisah agar nantinya bisa diekstrak.
- Semua REST response harus mengikuti `doc/api/response-standard.md`.
- Semua service harus mengikuti `doc/observability/tracing-and-idempotency.md`.

## 2. Pattern Wajib

- Outbox pattern untuk reliable event publishing.
- Inbox pattern atau processed-event table untuk idempotent consumer.
- Idempotency key untuk gRPC command yang mengubah state.
- Correlation ID pada semua REST request, gRPC metadata, log, dan Kafka event.
- Explicit state machine untuk order, payment, stock reservation, dan saga.
- Cache-aside pattern untuk Redis read cache.
- Database-backed idempotency harus tetap ada walaupun Redis lock digunakan.
- Page-based pagination untuk list REST endpoint.
- OpenTelemetry tracing untuk REST, gRPC, Kafka, PostgreSQL, Redis, dan Saga step.
- Kafka consumer harus mengimplementasikan inbox-based idempotency sebelum business mutation.

## 3. Jangan Generate

- Jangan buat shared database untuk semua service.
- Jangan membuat payment sebelum stock reservation sukses.
- Jangan langsung memanggil repository/database service lain.
- Jangan update stock di `order-service`.
- Jangan update order status langsung dari `payment-service`.
- Jangan memperlakukan Kafka event sebagai RPC response.
- Jangan hilangkan idempotency handling untuk duplicate event.
- Jangan taruh business logic di REST/gRPC/Kafka handler.
- Jangan import adapter dari domain package.
- Jangan gunakan Redis sebagai source of truth.
- Jangan gunakan cached stock untuk keputusan checkout final.
- Jangan mengembalikan bentuk REST response ad hoc.
- Jangan implementasikan list endpoint tanpa pagination.
- Jangan commit Kafka offset sebelum local transaction sukses.
- Jangan publish domain event di luar outbox pattern.
- Jangan drop `traceparent`, `correlation_id`, atau `causation_id` saat propagation REST/gRPC/Kafka.

## 4. Urutan Implementasi

Urutan coding yang direkomendasikan:

1. Definisikan shared contract dari dokumen:
   - OpenAPI
   - proto files
   - event envelope
2. Buat skeleton Go service menggunakan Hexagonal Architecture.
3. Implementasikan database migration per service.
4. Implementasikan Redis cache port dan adapter.
5. Implementasikan product read dan stock reservation di `catalog-inventory-service`.
6. Implementasikan create payment dan forced success/failure di `payment-service`.
7. Implementasikan create order dan saga orchestration di `order-service`.
8. Implementasikan outbox publisher worker.
9. Implementasikan Kafka consumer dan inbox table.
10. Implementasikan demo endpoint dan seed data.
11. Implementasikan test untuk success, insufficient stock, payment failure, cache fallback, dan duplicate event.

## 5. Aturan Coding

- Gunakan domain method eksplisit untuk state transition.
- Validasi state transition sebelum menerapkannya.
- Simpan product price snapshot di order item.
- Gunakan database transaction saat mengubah local state dan menulis outbox event.
- Retry transient failure dengan idempotency key yang sama.
- Buat structured log yang cukup untuk difilter berdasarkan `correlation_id` dan `order_id`.
- Jaga dependency package Go tetap mengarah ke dalam: adapter bergantung pada application/domain, bukan sebaliknya.
- Simpan generated gRPC code di luar domain package.
- Perlakukan Redis failure sebagai cache miss untuk non-critical read.
- Kembalikan `success`, `data` atau `error`, `meta`, dan `pagination` saat dibutuhkan.
- Sertakan `request_id`, `correlation_id`, dan `timestamp` di response metadata.
- Propagate OpenTelemetry context menggunakan REST headers, gRPC metadata, dan Kafka headers.
- Gunakan Kafka key `order_id` untuk event terkait checkout.

## 6. Error Handling

- Validation error harus mengembalikan HTTP 400.
- Missing resource harus mengembalikan HTTP 404.
- Business conflict seperti insufficient stock harus mengembalikan HTTP 409.
- Internal failure harus mengembalikan HTTP 500 atau gRPC internal error.
- Retryable downstream failure tidak boleh membuat duplicate business effect.

## 7. Adaptasi Library Go

Library Go spesifik boleh berbeda:

- HTTP framework.
- gRPC library.
- Kafka client.
- ORM/query builder.
- Migration tool.
- Logger/tracing library.
- Redis client.
- OpenTelemetry exporter/backend.

Pilihan berikut harus tetap stabil:

- Service boundary.
- Data ownership.
- API behavior.
- REST response envelope dan pagination behavior.
- Semantik gRPC command.
- Nama Kafka event dan makna payload.
- Saga state transition.
- Compensation rule.
- Arah dependency Hexagonal.
- Redis hanya cache.
- Semantik OpenTelemetry propagation.
- Semantik Kafka idempotent consumer.
