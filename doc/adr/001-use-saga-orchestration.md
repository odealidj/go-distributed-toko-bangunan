# ADR-001: Gunakan Saga Orchestration

## Status

Accepted

## Konteks

Checkout membutuhkan perubahan yang terkoordinasi di order, inventory, dan payment service. Satu ACID transaction tidak dapat menjangkau semua service ini tanpa menciptakan tight coupling dan kompleksitas operasional.

## Keputusan

Gunakan Saga orchestration untuk checkout distributed transaction.

Saga mengoordinasikan step berikut:

```text
Create Order -> Reserve Stock -> Create Payment -> Confirm Order -> Commit Stock
```

Compensation:

```text
Payment Failed -> Cancel Order -> Release Stock
```

## Konsekuensi

- Setiap service mempertahankan database sendiri.
- Cross-service consistency bersifat eventual.
- Compensation action harus idempotent.
- Saga state harus durable.
- Observability dibutuhkan untuk debug long-running flow.
