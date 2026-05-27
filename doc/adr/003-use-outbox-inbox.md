# ADR-003: Gunakan Outbox dan Inbox Pattern

## Status

Accepted

## Konteks

Kafka publishing dan local database write tidak dapat dibuat atomic dengan simple transaction. Kafka consumer juga dapat menerima duplicate event pada at-least-once delivery.

## Keputusan

Gunakan:

- Outbox pattern untuk reliable event publishing.
- Inbox pattern untuk idempotent event consumption.

Aturan producer:

```text
Mutate local state and insert outbox event in the same DB transaction.
```

Aturan consumer:

```text
Insert event_id into inbox_events before business mutation in the same DB transaction.
```

## Konsekuensi

- Duplicate Kafka delivery aman.
- Event dapat dipublish lebih dari sekali, tetapi consumer memproses business effect sekali.
- Outbox publisher worker dibutuhkan.
- Inbox table dibutuhkan pada service yang mengonsumsi event.
- Monitoring pending outbox row menjadi penting.
