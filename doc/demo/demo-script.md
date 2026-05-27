# Demo Script

## 1. Tujuan Demo

Tunjukkan bahwa sistem mini toko bangunan bukan sekadar CRUD. Demo ini menunjukkan:

- Go service dengan Hexagonal Architecture.
- Microservice boundary.
- gRPC synchronous call.
- Kafka asynchronous event.
- Redis cache yang tidak menggantikan durable state.
- Saga orchestration.
- Distributed transaction compensation.
- Idempotent consumer.

## 2. Service yang Ditunjukkan

- `order-service`: memiliki order dan mengorkestrasi checkout saga.
- `catalog-inventory-service`: memiliki catalog, stock, dan reservation.
- `payment-service`: memiliki simulasi payment.

## 3. Demo Data

Gunakan product berikut:

| Product | Unit | Price | Stock |
| --- | --- | --- | --- |
| Semen Tiga Roda 50kg | sak | 68000 | 20 |
| Besi Beton 10mm SNI | batang | 72000 | 15 |
| Pasir Bangka | m3 | 350000 | 5 |

## 4. Scenario A: Successful Checkout

Request:

```json
{
  "customer": {
    "name": "Budi",
    "phone": "08123456789",
    "address": "Jakarta"
  },
  "payment_mode": "SUCCESS",
  "items": [
    {
      "product_id": "prod_semen_50kg",
      "quantity": 2
    }
  ]
}
```

Talking point yang diharapkan:

- Order dibuat sebagai `PENDING`.
- Order service memanggil inventory via gRPC.
- Inventory melakukan reserve stock.
- Order service memanggil payment via gRPC.
- Payment mempublish `PaymentSucceeded`.
- Order service mengonsumsi event dan mengonfirmasi order.
- Inventory mengonsumsi `OrderConfirmed` dan melakukan commit stock.

Final state:

```text
Order: CONFIRMED
Payment: SUCCEEDED
Reservation: COMMITTED
Stock on hand reduced by 2
```

## 5. Scenario B: Insufficient Stock

Request:

```json
{
  "customer": {
    "name": "Sari",
    "phone": "08111111111",
    "address": "Bandung"
  },
  "payment_mode": "SUCCESS",
  "items": [
    {
      "product_id": "prod_pasir_m3",
      "quantity": 99
    }
  ]
}
```

Talking point yang diharapkan:

- Product valid, tetapi stock insufficient.
- Inventory menolak reservation.
- Order menjadi `REJECTED`.
- Payment tidak dibuat.

Final state:

```text
Order: REJECTED
Payment: not created
Reservation: FAILED or not created
Stock unchanged
```

## 6. Scenario C: Payment Failed dan Compensation

Request:

```json
{
  "customer": {
    "name": "Andi",
    "phone": "08222222222",
    "address": "Depok"
  },
  "payment_mode": "FAILURE",
  "items": [
    {
      "product_id": "prod_besi_10mm",
      "quantity": 3
    }
  ]
}
```

Talking point yang diharapkan:

- Stock awalnya di-reserve.
- Payment gagal.
- Order service menerima `PaymentFailed`.
- Order menjadi `CANCELLED`.
- Order service mempublish `OrderCancelled`.
- Inventory melakukan release reserved stock.

Final state:

```text
Order: CANCELLED
Payment: FAILED
Reservation: RELEASED
Available stock restored
```

## 7. Scenario D: Duplicate Event Idempotency

Action:

- Kirim ulang event `PaymentFailed` atau `OrderCancelled` yang sama dengan `event_id` yang sama.

Talking point yang diharapkan:

- Consumer mengecek `inbox_events`.
- Duplicate event diskip.
- Business state tidak berubah dua kali.

Final state:

```text
No double cancellation
No double stock release
No incorrect stock quantity
```

## 8. Urutan Presentasi yang Disarankan

1. Tunjukkan architecture diagram.
2. Tunjukkan service boundary.
3. Tunjukkan struktur package Go hexagonal.
4. Tunjukkan strategi Redis cache.
5. Tunjukkan kontrak gRPC proto.
6. Tunjukkan kontrak Kafka event.
7. Jalankan success scenario.
8. Jalankan insufficient stock scenario.
9. Jalankan payment failed compensation scenario.
10. Jalankan duplicate event/idempotency scenario.
11. Tutup dengan future extraction path untuk dedicated orchestrator.
