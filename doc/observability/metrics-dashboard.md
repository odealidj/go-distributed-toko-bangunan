# Metrics dan Resource Dashboard

## 1. Tujuan

Dokumen ini mendefinisikan metrics dan dashboard yang dibutuhkan untuk memantau performa sistem saat demo, integration test, dan K6 performance test.

K6 mengukur performa dari sisi client. Metrics dashboard mengukur kondisi internal sistem:

- CPU;
- RAM;
- container resource;
- PostgreSQL;
- Kafka;
- Redis;
- Go runtime;
- business/Saga metrics.

## 2. Stack Observability Metrics

Stack yang direkomendasikan untuk versi lengkap:

```text
Prometheus
Grafana
cAdvisor
node-exporter
postgres-exporter
redis-exporter
kafka-exporter
OpenTelemetry Collector
Jaeger
```

Peran:

| Komponen | Peran |
| --- | --- |
| Prometheus | Scrape dan menyimpan time-series metrics. |
| Grafana | Dashboard visualisasi. |
| cAdvisor | Container CPU/RAM/network/block I/O. |
| node-exporter | Host metrics. |
| postgres-exporter | PostgreSQL metrics. |
| redis-exporter | Redis metrics. |
| kafka-exporter | Kafka topic dan consumer group metrics. |
| OpenTelemetry Collector | Menerima trace/metrics/logs dari service. |
| Jaeger | Trace viewer. |

## 3. Minimum Stack untuk Portfolio

Jika ingin mulai sederhana:

```text
Prometheus
Grafana
cAdvisor
postgres-exporter
kafka-exporter
Jaeger
OpenTelemetry Collector
```

Redis exporter dapat ditambahkan setelah cache mulai aktif digunakan.

## 4. Service Metrics

Setiap Go service harus mengekspos:

```text
GET /metrics
```

Metric minimum:

| Metric | Type | Label |
| --- | --- | --- |
| `http_server_requests_total` | counter | service, route, method, status |
| `http_server_request_duration_seconds` | histogram | service, route, method |
| `grpc_server_requests_total` | counter | service, method, status |
| `grpc_server_request_duration_seconds` | histogram | service, method |
| `go_goroutines` | gauge | service |
| `go_memstats_heap_alloc_bytes` | gauge | service |
| `go_gc_duration_seconds` | summary/histogram | service |
| `process_cpu_seconds_total` | counter | service |
| `process_resident_memory_bytes` | gauge | service |

## 5. Business Metrics

Metric business/Saga yang direkomendasikan:

| Metric | Type | Label |
| --- | --- | --- |
| `checkout_orders_created_total` | counter | service |
| `checkout_orders_confirmed_total` | counter | service |
| `checkout_orders_cancelled_total` | counter | service, reason |
| `checkout_orders_rejected_total` | counter | service, reason |
| `checkout_saga_duration_seconds` | histogram | service, result |
| `checkout_saga_step_duration_seconds` | histogram | service, step, result |
| `stock_reservation_failed_total` | counter | service, reason |
| `payment_succeeded_total` | counter | service |
| `payment_failed_total` | counter | service, reason |

## 6. Outbox/Inbox Metrics

| Metric | Type | Label |
| --- | --- | --- |
| `outbox_pending_events` | gauge | service |
| `outbox_published_events_total` | counter | service, topic, event_type |
| `outbox_publish_failed_total` | counter | service, topic, event_type |
| `outbox_oldest_pending_age_seconds` | gauge | service |
| `inbox_processed_events_total` | counter | service, topic, event_type |
| `inbox_duplicate_events_total` | counter | service, topic, event_type |

Dashboard harus menunjukkan apakah outbox backlog naik lalu turun kembali setelah load test.

## 7. Kafka Metrics

Metric yang perlu dipantau:

- broker up/down;
- topic message in rate;
- topic bytes in/out;
- consumer group lag;
- partition count;
- failed produce/consume count dari service metrics;
- DLQ publish total.

Dashboard penting:

```text
consumer lag per consumer group
message rate per topic
DLQ count
```

Consumer group yang dipantau:

```text
order-service.payment-events-consumer
order-service.inventory-events-consumer
catalog-inventory-service.order-events-consumer
payment-service.order-events-consumer
```

## 8. PostgreSQL Metrics

Metric yang perlu dipantau:

- active connections;
- idle connections;
- query duration;
- transaction duration;
- locks;
- deadlocks;
- rows read/insert/update/delete;
- index hit ratio;
- database size;
- slow query count jika tersedia.

Untuk service pool:

```text
db_pool_open_connections
db_pool_in_use_connections
db_pool_idle_connections
db_pool_wait_count
db_pool_wait_duration_seconds
```

## 9. Redis Metrics

Metric yang perlu dipantau:

- used memory;
- memory fragmentation;
- connected clients;
- command duration;
- cache hits;
- cache misses;
- evicted keys;
- expired keys.

Custom metric dari service:

```text
redis_cache_hits_total
redis_cache_misses_total
redis_cache_errors_total
```

## 10. Container Resource Metrics

Dari cAdvisor:

- CPU usage per container;
- memory usage per container;
- network receive/transmit;
- block I/O;
- restart count;

Container yang wajib dipantau:

```text
order-service
catalog-inventory-service
payment-service
kafka
postgres
redis
```

## 11. Dashboard Grafana

Dashboard minimum:

1. API Overview
   - request rate;
   - p50/p95/p99 latency;
   - error rate.

2. Checkout Saga
   - created/confirmed/cancelled/rejected total;
   - saga duration;
   - step duration;
   - stuck saga.

3. Kafka
   - topic throughput;
   - consumer lag;
   - DLQ count.

4. Outbox/Inbox
   - pending event;
   - oldest pending age;
   - duplicate event count;
   - publish failure.

5. PostgreSQL
   - active connection;
   - query duration;
   - locks/deadlocks;
   - pool usage.

6. Resource
   - CPU/RAM per container;
   - Go goroutines;
   - heap;
   - GC.

## 12. Performance Test Correlation

Saat K6 berjalan, catat waktu mulai dan selesai:

```text
PERF_TEST_START
PERF_TEST_END
```

Gunakan annotation di Grafana jika memungkinkan.

Hal yang dilihat:

- Apakah p95 HTTP latency naik?
- Apakah Kafka consumer lag naik?
- Apakah outbox pending naik lalu turun?
- Apakah PostgreSQL connection pool jenuh?
- Apakah CPU/RAM service tertentu menjadi bottleneck?
- Apakah checkout terminal duration naik?

## 13. Resource Limit Untuk Demo

Untuk membuat bottleneck terlihat, Docker Compose dapat memberi resource limit.

Contoh konsep:

```yaml
cpus: 0.5
mem_limit: 256m
```

Catatan:

- dukungan resource limit tergantung Docker Compose runtime;
- jangan aktifkan limit terlalu kecil saat development biasa;
- gunakan profile khusus, misalnya `compose.perf.yml`.

## 14. Kriteria Sukses

Metrics/resource observability dianggap siap jika:

- Grafana dapat dibuka lokal;
- Prometheus berhasil scrape service dan exporter;
- CPU/RAM container terlihat;
- Kafka consumer lag terlihat;
- PostgreSQL connection dan query metrics terlihat;
- business metrics checkout terlihat;
- outbox/inbox metrics terlihat;
- K6 load test dapat dikorelasikan dengan dashboard.

