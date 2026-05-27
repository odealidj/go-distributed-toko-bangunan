# Desain Checkout Saga

## 1. Pattern

Checkout flow menggunakan Saga Orchestration. Untuk demo/MVP, `order-service` menjadi Saga orchestrator karena transaction lifecycle berpusat pada order.

Orchestrator diimplementasikan sebagai module terpisah di dalam `order-service` agar nantinya bisa diekstrak menjadi dedicated service.

## 2. Happy Path

```mermaid
sequenceDiagram
  participant FE as Frontend
  participant OS as order-service
  participant INV as catalog-inventory-service
  participant PAY as payment-service
  participant K as Kafka

  FE->>OS: POST /orders
  OS->>INV: ValidateProducts(items)
  INV-->>OS: valid product snapshots
  OS->>OS: Create order PENDING
  OS->>K: OrderCreated
  OS->>INV: ReserveStock(order_id, items)
  INV-->>OS: reservation success
  INV->>K: StockReserved
  OS->>OS: Update order STOCK_RESERVED
  OS->>PAY: CreatePayment(order_id, amount)
  PAY-->>OS: payment PENDING
  PAY->>K: PaymentCreated
  PAY->>K: PaymentSucceeded
  K-->>OS: PaymentSucceeded
  OS->>OS: Update order CONFIRMED
  OS->>K: OrderConfirmed
  K-->>INV: OrderConfirmed
  INV->>INV: Commit reserved stock
  INV->>K: StockCommitted
```

## 3. Payment Failed Path

```mermaid
sequenceDiagram
  participant OS as order-service
  participant INV as catalog-inventory-service
  participant PAY as payment-service
  participant K as Kafka

  OS->>INV: ReserveStock(order_id, items)
  INV-->>OS: reservation success
  OS->>PAY: CreatePayment(order_id, amount)
  PAY->>K: PaymentFailed
  K-->>OS: PaymentFailed
  OS->>OS: Update order CANCELLED
  OS->>K: OrderCancelled
  K-->>INV: OrderCancelled
  INV->>INV: Release reserved stock
  INV->>K: StockReleased
```

## 4. Insufficient Stock Path

```mermaid
sequenceDiagram
  participant FE as Frontend
  participant OS as order-service
  participant INV as catalog-inventory-service

  FE->>OS: POST /orders
  OS->>INV: ValidateProducts(items)
  INV-->>OS: valid product snapshots
  OS->>OS: Create order PENDING
  OS->>INV: ReserveStock(order_id, items)
  INV-->>OS: reservation failed
  OS->>OS: Update order REJECTED
```

## 5. Saga Steps

| Step | Owner | Action | Success | Failure | Compensation |
| --- | --- | --- | --- | --- | --- |
| 1 | order-service | Validate products | Product snapshot dikembalikan | Reject request/order | Tidak ada |
| 2 | order-service | Create order | Order `PENDING` tersimpan | Return error | Tidak ada |
| 3 | catalog-inventory-service | Reserve stock | `StockReserved` | `StockReservationFailed` | Tidak ada |
| 4 | payment-service | Create/process payment | `PaymentSucceeded` | `PaymentFailed` | Release stock |
| 5 | order-service | Confirm order | `OrderConfirmed` | Retry update | Release stock jika confirmation tidak mungkin |
| 6 | catalog-inventory-service | Commit stock | `StockCommitted` | Retry event handling | Manual repair jika gagal permanen |

## 6. Order State Machine

```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> STOCK_RESERVED: StockReserved
  PENDING --> REJECTED: StockReservationFailed
  STOCK_RESERVED --> WAITING_PAYMENT: PaymentCreated
  WAITING_PAYMENT --> CONFIRMED: PaymentSucceeded
  WAITING_PAYMENT --> CANCELLED: PaymentFailed
  STOCK_RESERVED --> CANCELLED: CancelRequested
  CONFIRMED --> [*]
  CANCELLED --> [*]
  REJECTED --> [*]
```

## 7. Aturan Compensation

- Jika payment gagal setelah stock berhasil di-reserve, order harus mempublish `OrderCancelled`.
- `catalog-inventory-service` harus release reservation ketika mengonsumsi `OrderCancelled`.
- Releasing stock harus idempotent.
- Committing stock harus idempotent.
- Jika duplicate `PaymentFailed` dikonsumsi, order harus tetap `CANCELLED` dan tidak boleh ada efek duplicate `OrderCancelled`.

## 8. Idempotency

Key yang wajib:

- gRPC commands: `idempotency_key`
- Kafka events: `event_id`
- Saga instance: `saga_id`
- Business aggregate: `order_id`

Format idempotency key yang disarankan:

```text
{operation}:{order_id}:{step_name}
```

Examples:

- `reserve-stock:ord_123:checkout`
- `create-payment:ord_123:checkout`
- `release-stock:ord_123:payment-failed`

## 9. Timeout dan Retry

- gRPC call harus menggunakan timeout pendek.
- Transient gRPC failure dapat diretry dengan idempotency key yang sama.
- Kafka consumer harus retry transient failure.
- Poison message pada akhirnya harus dipindahkan ke dead-letter topic jika DLQ diimplementasikan.
- Saga yang stuck di `STOCK_RESERVED` atau `WAITING_PAYMENT` melewati timeout konfigurasi harus ditandai untuk repair atau cancellation.

## 10. Jalur Extraction

Ketika dedicated orchestrator diperkenalkan:

- Pindahkan saga state dan orchestration module ke `checkout-orchestrator-service`.
- `order-service` mempublish `OrderCreated`.
- Orchestrator mengonsumsi `OrderCreated`, memanggil inventory/payment, lalu mempublish `CheckoutCompleted` atau `CheckoutFailed`.
- `order-service` mengonsumsi checkout result event dan mengupdate order status.
