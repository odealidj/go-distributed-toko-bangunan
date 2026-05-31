# Tracing, OpenTelemetry, dan Kafka Idempotency

## 1. Tujuan

Dokumen ini mendefinisikan bagaimana service menangani:

- Kafka idempotency.
- Request, correlation, causation, dan trace IDs.
- OpenTelemetry propagation.
- Log, metric, dan trace.

Tujuannya adalah membuat eksekusi checkout Saga observable dan aman terhadap retry, duplicate Kafka delivery, dan partial failure.

## 2. Tipe ID

| ID | Scope | Tujuan |
| --- | --- | --- |
| `request_id` | Satu HTTP request atau eksekusi internal handler | Debug satu request/handler attempt. |
| `correlation_id` | Satu business workflow | Menghubungkan semua log/event untuk satu checkout. |
| `causation_id` | Satu causal parent | Mengidentifikasi request/event yang menyebabkan event ini. |
| `trace_id` | Satu OpenTelemetry distributed trace | Menghubungkan span lintas REST, gRPC, Kafka, DB, dan Redis. |
| `event_id` | Satu domain event | Idempotency key untuk Kafka consumer. |
| `idempotency_key` | Satu command execution | Idempotency key untuk command yang mengubah state. |

Aturan:

- `request_id` dibuat untuk setiap incoming HTTP request dan eksekusi Kafka handler.
- `correlation_id` digunakan ulang di seluruh checkout Saga.
- `causation_id` diisi dengan parent `request_id` atau parent `event_id`.
- `trace_id` dikelola oleh OpenTelemetry melalui W3C Trace Context.
- `event_id` harus globally unique.
- `idempotency_key` harus stabil untuk retry command yang sama.

## 3. Keputusan OpenTelemetry

Semua Go service harus diinstrumentasi dengan OpenTelemetry.

Signal minimum yang wajib:

- Distributed trace.
- Structured log dengan trace context field.

Signal yang direkomendasikan:

- Metric untuk request duration, consumer lag, Kafka publish failure, outbox retry, dan Saga failure.

Pipeline local/demo:

```text
Go services -> OpenTelemetry Collector -> Jaeger or Grafana Tempo
```

## 4. Propagation

### REST Headers

Inbound REST harus membaca:

```text
traceparent
tracestate
X-Correlation-Id
X-Request-Id
```

Outbound REST response harus menyertakan:

```text
X-Correlation-Id
X-Request-Id
```

Response body `meta` harus menyertakan:

```json
{
  "request_id": "req_123",
  "correlation_id": "corr_123",
  "timestamp": "2026-05-27T10:00:00Z"
}
```

### gRPC Metadata

gRPC client harus mempropagasi:

```text
traceparent
tracestate
x-correlation-id
x-request-id
x-causation-id
x-idempotency-key
```

gRPC command yang mengubah state harus menyertakan `idempotency_key` di request body atau metadata. Proto request metadata sudah menyertakan field ini untuk inventory dan payment command.

### Kafka Headers

Kafka producer harus mengisi header berikut:

```text
traceparent
tracestate
x-correlation-id
x-causation-id
x-event-id
x-event-type
```

Kafka event payload juga harus menyertakan:

```text
event_id
event_type
correlation_id
causation_id
```

Header digunakan untuk transport propagation. Payload field digunakan untuk domain audit dan replay.

## 5. Kafka Delivery Model

Aplikasi mengasumsikan Kafka delivery adalah at-least-once.

Karena itu:

- Duplicate event adalah kondisi yang diharapkan.
- Consumer harus idempotent.
- Business effect harus terjadi exactly once pada application level.
- Kafka offset hanya boleh dicommit setelah local transaction sukses.

Jangan bergantung pada Kafka exactly-once semantics sebagai satu-satunya perlindungan untuk business correctness.

## 6. Algoritma Idempotent Consumer

Setiap Kafka consumer yang mengubah local state harus mengikuti algoritma ini:

```text
1. Terima Kafka event.
2. Extract event_id, event_type, aggregate_id, correlation_id, traceparent.
3. Mulai OpenTelemetry consumer span.
4. Mulai local DB transaction.
5. Insert event_id ke inbox_events.
6. Jika event_id sudah ada:
   a. Rollback atau commit no-op.
   b. Tandai event sebagai duplicate di log/metric.
   c. Commit Kafka offset.
   d. Hentikan processing.
7. Jalankan business state transition.
8. Tulis outbox event jika transition menghasilkan event baru.
9. Commit local DB transaction.
10. Commit Kafka offset.
```

Kolom `inbox_events.event_id` harus memiliki unique constraint.

## 7. Algoritma Outbox Publisher

Setiap state mutation yang menghasilkan event harus menggunakan outbox:

```text
1. Mulai local DB transaction.
2. Mutate local business state.
3. Insert event envelope ke outbox_events dengan status PENDING.
4. Commit local DB transaction.
5. Outbox worker membaca row PENDING.
6. Worker mempublish event ke Kafka dengan header wajib.
7. Worker menandai row sebagai PUBLISHED.
```

Jika publish sukses tetapi marking `PUBLISHED` gagal, event dapat dipublish ulang. Consumer harus menangani ini melalui inbox idempotency.

## 8. Kafka Message Key

Event terkait checkout harus menggunakan:

```text
Kafka key = order_id
```

Alasan:

- Event untuk order yang sama masuk ke partition yang sama.
- Relative ordering untuk satu order terjaga.
- Debugging Saga flow lebih mudah.

## 9. Kebutuhan Span

Setiap service harus membuat span untuk:

- REST inbound request.
- gRPC inbound request.
- gRPC outbound call.
- Kafka publish.
- Kafka consume.
- PostgreSQL query/transaction.
- Redis command.
- Application use case.
- Saga step.

Nama span yang direkomendasikan:

```text
POST /orders
OrderUseCase.CreateOrder
CheckoutSaga.ReserveStock
InventoryService/ReserveStock
PaymentService/CreatePayment
Kafka publish payment.events PaymentSucceeded
Kafka consume payment.events PaymentSucceeded
```

## 10. Span Attribute yang Wajib

Gunakan attribute ini jika berlaku:

| Attribute | Example |
| --- | --- |
| `service.name` | `order-service` |
| `correlation_id` | `corr_123` |
| `request_id` | `req_123` |
| `order_id` | `ord_123` |
| `event_id` | `evt_123` |
| `event_type` | `PaymentSucceeded` |
| `saga_id` | `saga_123` |
| `saga_step` | `reserve_stock` |
| `messaging.system` | `kafka` |
| `messaging.destination.name` | `payment.events` |
| `db.system` | `postgresql` |
| `db.operation.name` | `INSERT` |
| `cache.system` | `redis` |

## 11. Aturan Logging

Log harus structured dan menyertakan:

```text
timestamp
level
service
message
request_id
correlation_id
trace_id
span_id
order_id
event_id
event_type
```

Jangan log:

- raw password
- payment secret
- full access token
- internal stack trace pada client response

Implementasi local saat ini:

- semua service menginisialisasi OpenTelemetry tracer provider ke OTLP gRPC endpoint dari `OTEL_EXPORTER_OTLP_ENDPOINT`;
- logger default menggunakan `slog` JSON handler yang otomatis menambahkan `service`, `request_id`, `correlation_id`, `trace_id`, dan `span_id` dari `context.Context`;
- Kafka producer menginjeksi `traceparent` melalui OpenTelemetry propagator ke header record;
- Kafka consumer mengekstrak trace context dari header lalu membuat span `Kafka consume ...`;
- `order-service` menyimpan `traceparent` ke `outbox_events` agar publish async tetap berada dalam trace yang sama;
- `catalog-inventory-service` dan `payment-service` memproses event Kafka dengan `slog.InfoContext(...)` agar log terikat ke trace dan correlation yang aktif.

## 12. Metrics

Metric yang direkomendasikan:

| Metric | Type | Labels |
| --- | --- | --- |
| `http_server_requests_total` | counter | service, route, method, status |
| `http_server_request_duration_seconds` | histogram | service, route, method |
| `grpc_server_requests_total` | counter | service, method, status |
| `kafka_events_consumed_total` | counter | service, topic, event_type, result |
| `kafka_events_published_total` | counter | service, topic, event_type, result |
| `kafka_event_duplicates_total` | counter | service, topic, event_type |
| `outbox_pending_total` | gauge | service |
| `outbox_publish_failures_total` | counter | service, topic |
| `saga_transitions_total` | counter | service, saga_step, result |
| `saga_stuck_total` | gauge | service |
| `redis_cache_hits_total` | counter | service, cache_name |
| `redis_cache_misses_total` | counter | service, cache_name |

## 13. Failure Handling

### Duplicate Event

Perilaku yang diharapkan:

- Deteksi melalui `inbox_events.event_id`.
- Skip business mutation.
- Commit Kafka offset.
- Naikkan duplicate metric.

### Poison Event

Perilaku yang diharapkan:

- Retry transient failure dengan backoff.
- Setelah retry limit konfigurasi tercapai, kirim ke DLQ jika diimplementasikan.
- Simpan metadata yang cukup untuk replay setelah repair.

### Trace Context Hilang

Perilaku yang diharapkan:

- Generate trace context baru.
- Pertahankan/generate `correlation_id`.
- Lanjutkan processing.

## 14. Demo Observability Flow

Demo harus menunjukkan satu trace untuk:

```text
POST /orders
-> ValidateProducts gRPC
-> ReserveStock gRPC
-> CreatePayment gRPC
-> PaymentSucceeded Kafka publish
-> PaymentSucceeded Kafka consume
-> OrderConfirmed Kafka publish
-> OrderConfirmed Kafka consume
-> StockCommitted Kafka publish
```
