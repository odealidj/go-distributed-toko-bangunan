# ADR-006: Gunakan Redis Hanya Sebagai Cache

## Status

Accepted

## Konteks

Redis meningkatkan read performance dan dapat membantu short-lived lock. Namun, checkout correctness bergantung pada durable state dan recovery setelah partial failure.

## Keputusan

Gunakan Redis sebagai cache dan optional short-lived coordination helper, bukan sebagai source of truth.

Redis boleh menyimpan:

- Product detail cache.
- Product list cache.
- Order detail cache.
- Payment status cache.
- Short-lived idempotency or Saga locks.

Redis tidak boleh menyimpan satu-satunya copy dari:

- Orders.
- Payments.
- Inventory.
- Stock reservations.
- Saga state.
- Outbox/inbox state.

## Konsekuensi

- Redis failure seharusnya menurunkan performance, bukan correctness.
- Keputusan checkout kritikal harus read/write PostgreSQL.
- Cache invalidation dibutuhkan setelah perubahan product, stock display, order, atau payment.
- Database-backed idempotency tetap wajib.
