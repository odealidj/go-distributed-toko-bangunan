# Logical Data Model

## 1. Aturan Umum

- Setiap service memiliki datanya sendiri.
- Tidak ada service yang boleh query langsung ke database service lain.
- Referensi lintas service disimpan sebagai ID dan snapshot.
- Order item menyimpan product name, unit, dan price snapshot untuk menjaga historical correctness.
- Semua service harus memiliki tabel `outbox_events` dan `inbox_events` atau mekanisme durable yang setara.
- Redis bukan bagian dari durable logical data model.
- Redis key boleh menyimpan cache read model, tetapi tabel PostgreSQL tetap menjadi source of truth.

## 2. Order Service

### orders

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| customer_name | string | Snapshot. |
| customer_phone | string | Snapshot. |
| customer_address | string | Optional snapshot. |
| status | enum | Lihat `order_status`. |
| total_amount | integer | Unit mata uang terkecil. |
| payment_id | string | Nullable. |
| correlation_id | string | Trace ID. |
| created_at | timestamp | Wajib. |
| updated_at | timestamp | Wajib. |

### order_items

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| order_id | string | FK ke tabel orders lokal. |
| product_id | string | Referensi eksternal ke catalog inventory. |
| product_name | string | Snapshot. |
| unit | string | Snapshot. |
| quantity | decimal | Wajib. |
| unit_price | integer | Snapshot. |
| line_total | integer | Wajib. |

### saga_instances

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| order_id | string | Unique. |
| status | enum | Saga status. |
| current_step | string | Step terakhir yang completed/active. |
| correlation_id | string | Trace ID. |
| started_at | timestamp | Wajib. |
| completed_at | timestamp | Nullable. |

### saga_steps

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| saga_id | string | FK ke saga_instances. |
| step_name | string | Example: `reserve_stock`. |
| status | enum | `PENDING`, `SUCCEEDED`, `FAILED`, `COMPENSATED`. |
| idempotency_key | string | Unique untuk command step. |
| error_message | string | Nullable. |
| created_at | timestamp | Wajib. |
| updated_at | timestamp | Wajib. |

## 3. Catalog Inventory Service

### categories

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| name | string | Contoh: Semen, Besi, Cat. |
| is_active | boolean | Wajib. |

### products

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| category_id | string | Local FK. |
| sku | string | Unique. |
| name | string | Wajib. |
| brand | string | Optional. |
| unit | string | Example: `sak`, `batang`, `m3`. |
| price | integer | Harga jual saat ini. |
| weight_kg | decimal | Optional. |
| requires_truck | boolean | Wajib. |
| is_active | boolean | Wajib. |

### inventories

| Column | Type | Catatan |
| --- | --- | --- |
| product_id | string | Primary key dan FK ke products. |
| on_hand_qty | decimal | Physical stock. |
| reserved_qty | decimal | Stock yang sedang reserved. |
| updated_at | timestamp | Wajib. |

### stock_reservations

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| order_id | string | Unique per successful reservation. |
| status | enum | `RESERVED`, `COMMITTED`, `RELEASED`, `FAILED`. |
| idempotency_key | string | Unique. |
| created_at | timestamp | Wajib. |
| updated_at | timestamp | Wajib. |

### stock_reservation_items

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| reservation_id | string | FK ke stock_reservations. |
| product_id | string | FK ke products. |
| quantity | decimal | Wajib. |

## 4. Payment Service

### payments

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| order_id | string | Unique. |
| amount | integer | Wajib. |
| status | enum | `PENDING`, `SUCCEEDED`, `FAILED`, `CANCELLED`. |
| payment_mode | enum | `SUCCESS`, `FAILURE`, `MANUAL`. |
| idempotency_key | string | Unique. |
| created_at | timestamp | Wajib. |
| updated_at | timestamp | Wajib. |

### payment_attempts

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Primary key. |
| payment_id | string | FK ke payments. |
| status | enum | Attempt result. |
| reason | string | Nullable. |
| created_at | timestamp | Wajib. |

## 5. Tabel Infrastruktur Bersama

### outbox_events

| Column | Type | Catatan |
| --- | --- | --- |
| id | string | Event ID. |
| aggregate_id | string | Biasanya order_id. |
| aggregate_type | string | Contoh: order, payment, stock_reservation. |
| event_type | string | Wajib. |
| correlation_id | string | Business workflow ID. |
| causation_id | string | Parent request/event ID. |
| traceparent | string | W3C trace context saat publish. |
| payload | json | Event envelope. |
| status | enum | `PENDING`, `PUBLISHED`, `FAILED`. |
| retry_count | integer | Wajib. |
| created_at | timestamp | Wajib. |
| published_at | timestamp | Nullable. |

### inbox_events

| Column | Type | Catatan |
| --- | --- | --- |
| event_id | string | Primary key. |
| event_type | string | Wajib. |
| aggregate_id | string | Wajib. |
| correlation_id | string | Business workflow ID. |
| traceparent | string | W3C trace context yang diterima dari Kafka headers. |
| processed_at | timestamp | Wajib. |

Aturan processing:

```text
Insert ke inbox_events harus terjadi dalam local transaction yang sama dengan business mutation yang disebabkan oleh event.
```

## 6. Redis Cache Keys

Redis key diturunkan dari durable data dan dapat dihapus kapan saja tanpa kehilangan data.

Examples:

| Service | Key Pattern | Source of Truth |
| --- | --- | --- |
| order-service | `order-service:order:{order_id}` | `orders`, `order_items` |
| order-service | `order-service:lock:saga:{order_id}` | `saga_instances`, `saga_steps` |
| catalog-inventory-service | `catalog-inventory-service:product:{product_id}` | `products`, `inventories` |
| catalog-inventory-service | `catalog-inventory-service:products:list:{filter_hash}` | `products`, `categories`, `inventories` |
| payment-service | `payment-service:payment:{payment_id}` | `payments`, `payment_attempts` |

Aturan cache correctness:

```text
Jika Redis kosong atau tidak tersedia, sistem tetap harus menghasilkan business result yang benar dari PostgreSQL.
```
