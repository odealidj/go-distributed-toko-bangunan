# Kafka Operational Design

## 1. Tujuan

Dokumen ini mendefinisikan keputusan operasional Kafka untuk local development dan demo production-like.

## 2. Broker

Gunakan Apache Kafka dalam KRaft mode.

Tidak menggunakan Zookeeper.

Listener:

```text
kafka:9092       # internal Docker network
localhost:29092 # akses dari host
```

## 3. Kafka Client

Gunakan:

```text
github.com/twmb/franz-go/pkg/kgo
```

Aturan:

- producer menggunakan required Kafka headers;
- consumer menggunakan manual offset commit;
- offset commit hanya dilakukan setelah local transaction sukses;
- message key untuk event checkout adalah `order_id`.

## 4. Topics

Topic utama:

| Topic | Producer | Consumer | Partitions | Replication |
| --- | --- | --- | --- | --- |
| `order.events` | order-service | catalog-inventory-service, payment-service | 3 | 1 |
| `inventory.events` | catalog-inventory-service | order-service | 3 | 1 |
| `payment.events` | payment-service | order-service | 3 | 1 |

DLQ topic:

| Topic | Purpose | Partitions | Replication |
| --- | --- | --- | --- |
| `order.events.dlq` | Failed order event processing | 3 | 1 |
| `inventory.events.dlq` | Failed inventory event processing | 3 | 1 |
| `payment.events.dlq` | Failed payment event processing | 3 | 1 |

Local demo menggunakan replication factor `1` karena hanya ada satu broker.

## 5. Message Key

Semua event terkait checkout harus menggunakan:

```text
key = order_id
```

Alasan:

- event untuk order yang sama masuk partition yang sama;
- ordering relatif per order lebih mudah dijaga;
- trace/debug Saga lebih mudah.

## 6. Consumer Group

Gunakan consumer group per service dan purpose:

```text
order-service.payment-events-consumer
order-service.inventory-events-consumer
catalog-inventory-service.order-events-consumer
payment-service.order-events-consumer
```

Aturan:

- satu consumer group mewakili satu logical processor;
- jangan berbagi group untuk use case berbeda;
- consumer group name harus stabil agar offset tidak hilang.

## 7. Offset Commit

Gunakan manual offset commit.

Urutan wajib:

```text
1. Consume event.
2. Begin local DB transaction.
3. Insert event_id ke inbox_events.
4. Jika duplicate, skip mutation.
5. Jalankan business mutation jika belum duplicate.
6. Tulis outbox event jika diperlukan.
7. Commit local DB transaction.
8. Commit Kafka offset.
```

Jangan commit offset sebelum local transaction sukses.

## 8. Retry dan DLQ

Retry transient error dengan backoff.

Jika retry limit tercapai:

```text
publish ke {source-topic}.dlq
commit offset original event
```

DLQ payload harus menyertakan:

- original topic;
- original partition;
- original offset;
- original key;
- original headers;
- original payload;
- error message;
- failed service;
- failed at timestamp.

## 9. Header Wajib

Kafka producer wajib mengirim:

```text
traceparent
tracestate
x-correlation-id
x-causation-id
x-event-id
x-event-type
```

Detail ada di:

```text
doc/observability/tracing-and-idempotency.md
```

## 10. Retention

Untuk local/demo:

```text
retention.ms = 604800000 # 7 hari
```

Untuk production-like kecil:

```text
retention.ms = 604800000 sampai 1209600000 # 7-14 hari
```

Outbox table tetap menjadi audit lokal event yang dipublish.

## 11. Topic Creation

Topic dibuat eksplisit melalui Make target:

```text
make kafka-topics
```

Command contoh:

```text
kafka-topics --bootstrap-server kafka:9092 --create --topic order.events --partitions 3 --replication-factor 1
```

Untuk local Compose, topic yang dibuat adalah:

- `order.events`
- `inventory.events`
- `payment.events`

Pada phase 05, `order-service` mempublish event dari `outbox_events` ke `order.events`.
`catalog-inventory-service` dan `payment-service` mengonsumsi `order.events` memakai manual offset commit setelah event berhasil masuk `inbox_events` dan business mutation selesai.

## 12. Observability

Metric/log minimal:

- consumer lag;
- event consumed total;
- event published total;
- duplicate event total;
- DLQ publish total;
- outbox pending total;
- publish error total.

Kafka UI digunakan untuk melihat:

- topic;
- partition;
- consumer group;
- lag;
- message sample.

## 13. Batasan Local

Local Kafka single broker tidak merepresentasikan high availability production.

Yang tetap bisa dipelajari:

- topic;
- partition;
- consumer group;
- offset;
- message key;
- retry;
- DLQ;
- idempotent consumer;
- outbox publisher.
