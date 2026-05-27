# ADR-005: Gunakan Kafka untuk Domain Event

## Status

Accepted

## Konteks

Checkout menghasilkan domain event yang harus diproses secara asinkron, seperti payment result, order confirmation, cancellation, stock commit, dan stock release.

## Keputusan

Gunakan Kafka untuk domain event.

Topics:

```text
order.events
inventory.events
payment.events
```

Kafka message key untuk event terkait checkout:

```text
order_id
```

## Konsekuensi

- Service menjadi loosely coupled melalui event.
- Event consumer harus idempotent.
- Event contract harus stabil dan diversi dengan hati-hati.
- Outbox/inbox pattern wajib.
- Local demo dapat menggunakan single-node Kafka atau Redpanda untuk mengurangi operational cost.
