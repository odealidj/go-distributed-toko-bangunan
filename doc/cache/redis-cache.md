# Strategi Redis Cache

## 1. Keputusan

Redis digunakan sebagai cache dan short-lived coordination helper. PostgreSQL tetap menjadi source of truth untuk semua business data.

Redis tidak boleh dibutuhkan untuk merekonstruksi order, payment, inventory, outbox, atau inbox state.

## 2. Cache Ownership

Setiap service memiliki cache key-nya sendiri.

Diizinkan:

```text
order-service -> order-related Redis keys
catalog-inventory-service -> product/catalog Redis keys
payment-service -> payment-related Redis keys
```

Dilarang:

```text
payment-service reading order-service Redis keys
order-service reading inventory Redis keys directly
```

Akses data lintas service tetap harus menggunakan REST, gRPC, atau Kafka.

## 3. Penggunaan yang Direkomendasikan per Service

### order-service

Gunakan Redis untuk:

- short-lived order detail read cache
- optional idempotency request lock for `POST /orders`
- optional Saga processing lock per `order_id`

Jangan gunakan Redis untuk:

- order source of truth
- Saga durable state
- outbox/inbox durability

### catalog-inventory-service

Gunakan Redis untuk:

- product detail cache
- product list cache
- category cache

Berhati-hati dengan:

- stock availability cache

Stock yang ditampilkan di catalog boleh eventually consistent, tetapi checkout harus bergantung pada `ReserveStock` terhadap PostgreSQL primary/source of truth.

### payment-service

Gunakan Redis untuk:

- short-lived payment status cache
- optional idempotency lock for `CreatePayment`

Jangan gunakan Redis untuk:

- final payment state
- payment attempt history

## 4. Penamaan Key

Format yang direkomendasikan:

```text
{service}:{domain}:{identifier}
```

Examples:

```text
catalog-inventory-service:product:prod_semen_50kg
catalog-inventory-service:products:list:category:semen
order-service:order:ord_123
order-service:lock:saga:ord_123
payment-service:payment:pay_123
```

## 5. Rekomendasi TTL

| Cache | TTL |
| --- | --- |
| Product detail | 5-15 minutes |
| Product list | 1-5 minutes |
| Category list | 15-60 minutes |
| Order detail | 30-120 seconds |
| Payment status | 30-120 seconds |
| Idempotency lock | 30-300 seconds |
| Saga lock | 30-300 seconds |

## 6. Aturan Invalidation

- Product update harus menghapus product detail dan product list cache terkait.
- Inventory stock mutation dapat menghapus product detail/list cache jika stock ditampilkan.
- Order status transition harus menghapus order detail cache.
- Payment status transition harus menghapus payment status cache.

Untuk demo, direct cache deletion sudah cukup. Untuk production, cache invalidation juga dapat didorong oleh domain event.

## 7. Aturan Consistency

- Cache miss membaca dari PostgreSQL lalu menyimpan ke Redis.
- Cache failure tidak boleh menggagalkan critical write operation.
- Keputusan checkout kritikal tidak boleh bergantung pada cached stock.
- gRPC command yang mengubah state harus bypass cache untuk validasi yang memengaruhi correctness.

## 8. Perilaku Saat Redis Failure

Jika Redis tidak tersedia:

- catalog read harus fallback ke PostgreSQL.
- order read harus fallback ke PostgreSQL.
- payment read harus fallback ke PostgreSQL.
- checkout flow harus tetap berjalan jika Redis hanya digunakan sebagai cache.
- optional lock harus memiliki database idempotency sebagai durable fallback.
