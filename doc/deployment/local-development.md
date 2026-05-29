# Panduan Local Development

## 1. Komponen yang Dibutuhkan

Local environment harus menyediakan:

- Go toolchain.
- Podman dan `podman compose` untuk local development.
- Apache Kafka broker dengan KRaft mode.
- PostgreSQL databases, one per service.
- Redis.
- OpenTelemetry Collector.
- Jaeger or Grafana Tempo for trace viewing.
- `order-service`.
- `catalog-inventory-service`.
- `payment-service`.
- Optional Kafka UI.

## 2. Port yang Disarankan

| Component | Port |
| --- | --- |
| order-service REST | 8080 |
| catalog-inventory-service REST | 8081 |
| order-service gRPC | 9000 |
| catalog-inventory-service gRPC | 9001 |
| payment-service gRPC | 9002 |
| Kafka | 9092 |
| Kafka host listener | 29092 |
| Kafka UI | 8090 |
| Redis | 6379 |
| OpenTelemetry Collector OTLP gRPC | 4317 |
| OpenTelemetry Collector OTLP HTTP | 4318 |
| Jaeger UI | 16686 |
| PostgreSQL order | 5433 |
| PostgreSQL inventory | 5434 |
| PostgreSQL payment | 5435 |

## 3. Environment Variable yang Dibutuhkan

### order-service

```text
SERVICE_NAME=order-service
HTTP_ADDR=:8080
GRPC_ADDR=:9000
DATABASE_URL=postgres://order:order@localhost:5433/order_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_KEY_PREFIX=order-service
OTEL_SERVICE_NAME=order-service
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
KAFKA_BROKERS=localhost:9092
INVENTORY_GRPC_ADDR=localhost:9001
PAYMENT_GRPC_ADDR=localhost:9002
```

### catalog-inventory-service

```text
SERVICE_NAME=catalog-inventory-service
HTTP_ADDR=:8081
GRPC_ADDR=:9001
DATABASE_URL=postgres://inventory:inventory@localhost:5434/inventory_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_KEY_PREFIX=catalog-inventory-service
OTEL_SERVICE_NAME=catalog-inventory-service
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
KAFKA_BROKERS=localhost:9092
```

### payment-service

```text
SERVICE_NAME=payment-service
GRPC_ADDR=:9002
DATABASE_URL=postgres://payment:payment@localhost:5435/payment_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_KEY_PREFIX=payment-service
OTEL_SERVICE_NAME=payment-service
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
KAFKA_BROKERS=localhost:9092
```

## 4. Seed Data

Produk demo yang direkomendasikan:

| Product ID | Name | Unit | Price | On Hand |
| --- | --- | --- | --- | --- |
| `prod_semen_50kg` | Semen Portland 50kg | sak | 68000 | 120 |
| `prod_besi_10mm` | Besi Beton 10mm | batang | 72000 | 80 |
| `prod_pasir_1m3` | Pasir Beton 1m3 | m3 | 310000 | 12 |
| `prod_cat_putih_5kg` | Cat Tembok Putih 5kg | pail | 145000 | 45 |

## 5. Urutan Startup

1. Jalankan PostgreSQL database.
2. Jalankan Apache Kafka KRaft.
3. Jalankan Redis.
4. Jalankan OpenTelemetry Collector dan Jaeger/Tempo.
5. Jalankan migration untuk setiap service.
6. Jalankan seed data catalog dan inventory.
7. Jalankan `catalog-inventory-service`.
8. Jalankan `payment-service`.
9. Jalankan `order-service`.
10. Jalankan demo scenario.

## 6. Health Checks

Setiap service harus mengekspos:

```text
GET /healthz
GET /readyz
```

Readiness harus memverifikasi dependency yang dibutuhkan:

- Database connection.
- Redis connection harus dicek, tetapi cache failure tidak boleh merusak durable state.
- Kafka producer/consumer readiness jika berlaku.
- gRPC downstream yang dibutuhkan untuk `order-service`.
- OpenTelemetry exporter misconfiguration harus dilog, tetapi tidak boleh memblokir core business flow dalam demo mode.

## 7. Catatan Docker Compose

Detail topology Docker Compose dijelaskan di:

```text
doc/deployment/docker-compose-architecture.md
```

Untuk service di dalam Docker network:

```text
KAFKA_BROKERS=kafka:9092
```

Untuk tool dari host:

```text
localhost:29092
```

Local default menggunakan Podman:

```text
COMPOSE="podman compose"
```

Jika dijalankan di VPS dengan Docker:

```text
make infra-up COMPOSE="docker compose"
make up COMPOSE="docker compose"
```

## 8. Workflow Debug Lokal

Untuk debug service dari IDE/local terminal:

```text
make infra-up
make kafka-topics
make inventory-run
make payment-run
make order-run
```

`make infra-up` hanya menjalankan infrastructure:

```text
postgres
redis
kafka
kafka-ui
otel-collector
jaeger
prometheus/grafana jika profile observability aktif
```

Service aplikasi dapat dijalankan terpisah agar breakpoint/debugger lebih mudah digunakan.

## 9. Verifikasi Trace Demo

Setelah semua service jalan dan seed data sudah masuk:

```text
make trace-verify
```

Target ini akan:

1. melakukan checkout demo ke `order-service`;
2. menunggu trace dikirim ke Jaeger;
3. mencari trace berdasarkan `correlation_id`;
4. memverifikasi minimal ada `order-service`, `catalog-inventory-service`, dan `payment-service` di trace yang sama.

Script yang dipakai:

```text
scripts/verify-trace.sh
```
