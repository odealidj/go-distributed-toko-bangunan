# Kontrak Event

## 1. Envelope

Semua Kafka message harus menggunakan envelope ini:

```json
{
  "event_id": "evt_123",
  "event_type": "PaymentSucceeded",
  "aggregate_id": "ord_123",
  "aggregate_type": "order",
  "occurred_at": "2026-05-27T10:00:00Z",
  "correlation_id": "corr_123",
  "causation_id": "evt_previous",
  "trace_id": "otel_trace_id_optional_copy",
  "payload": {}
}
```

Aturan:

- `event_id` harus globally unique.
- Kafka message key sebaiknya `order_id`.
- `aggregate_id` sebaiknya `order_id` untuk event terkait checkout.
- Consumer harus menyimpan `event_id` yang sudah diproses.
- Event schema bersifat append-only. Jangan hapus atau rename field yang sudah ada.
- Kafka headers harus membawa OpenTelemetry trace context.
- Event payload menyimpan correlation dan causation field untuk domain audit/replay.

## 2. Kafka Headers yang Wajib

Setiap Kafka message yang diproduksi harus menyertakan:

| Header | Deskripsi |
| --- | --- |
| `traceparent` | W3C Trace Context yang dipropagasi oleh OpenTelemetry. |
| `tracestate` | Optional W3C Trace Context vendor state. |
| `x-correlation-id` | Business workflow correlation ID. |
| `x-causation-id` | Parent request ID atau event ID. |
| `x-event-id` | Value yang sama dengan payload `event_id`. |
| `x-event-type` | Value yang sama dengan payload `event_type`. |

Kafka message key untuk event terkait checkout:

```text
key = order_id
```

## 3. Aturan Idempotent Consumer

Consumer harus mengasumsikan at-least-once delivery.

Urutan processing yang wajib:

```text
1. Mulai local DB transaction.
2. Insert event_id ke inbox_events.
3. Jika event_id sudah ada, skip business mutation.
4. Terapkan business mutation.
5. Tulis outbox event jika dibutuhkan.
6. Commit local DB transaction.
7. Commit Kafka offset.
```

Kolom `inbox_events.event_id` harus unique.

## 4. Topics

| Topic | Producer | Consumer |
| --- | --- | --- |
| `order.events` | order-service | catalog-inventory-service, payment-service |
| `inventory.events` | catalog-inventory-service | order-service |
| `payment.events` | payment-service | order-service |

## 5. Order Events

### OrderCreated

Producer: `order-service`

Consumer: optional audit consumer, future orchestrator

Payload:

```json
{
  "order_id": "ord_123",
  "status": "PENDING",
  "total_amount": 215000,
  "items": [
    {
      "product_id": "prod_semen_50kg",
      "product_name": "Semen Tiga Roda 50kg",
      "unit": "sak",
      "quantity": 2,
      "unit_price": 68000,
      "line_total": 136000
    }
  ]
}
```

### OrderConfirmed

Producer: `order-service`

Consumer: `catalog-inventory-service`

Efek:

- Inventory melakukan commit reserved stock.

### OrderCancelled

Producer: `order-service`

Consumer: `catalog-inventory-service`, `payment-service`

Efek:

- Inventory melakukan release reserved stock.
- Payment membatalkan pending payment jika memungkinkan.

### OrderRejected

Producer: `order-service`

Consumer: optional audit consumer

Efek:

- Menandakan checkout tidak dapat dilanjutkan, umumnya karena stock reservation gagal.

## 6. Inventory Events

### StockReserved

Producer: `catalog-inventory-service`

Consumer: `order-service`

Efek:

- Order dapat transition ke `STOCK_RESERVED`.
- Orchestrator dapat melanjutkan ke payment creation.

### StockReservationFailed

Producer: `catalog-inventory-service`

Consumer: `order-service`

Efek:

- Order transition ke `REJECTED`.
- Payment tidak boleh dibuat.

### StockCommitted

Producer: `catalog-inventory-service`

Consumer: optional audit consumer

Efek:

- Reservation sudah dikonversi menjadi final stock deduction.

### StockReleased

Producer: `catalog-inventory-service`

Consumer: optional audit consumer

Efek:

- Reserved stock sudah dikembalikan ke available stock.

## 7. Payment Events

### PaymentCreated

Producer: `payment-service`

Consumer: `order-service`

Efek:

- Order dapat transition ke `WAITING_PAYMENT`.

### PaymentSucceeded

Producer: `payment-service`

Consumer: `order-service`

Efek:

- Order transition ke `CONFIRMED`.
- Order mempublish `OrderConfirmed`.

### PaymentFailed

Producer: `payment-service`

Consumer: `order-service`

Efek:

- Order transition ke `CANCELLED`.
- Order mempublish `OrderCancelled`.

### PaymentCancelled

Producer: `payment-service`

Consumer: optional audit consumers

Efek:

- Payment sudah dibatalkan karena order dibatalkan.
