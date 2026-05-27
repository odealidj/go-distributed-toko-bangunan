# Keputusan Teknologi

## 1. Tujuan

Dokumen ini mengunci pilihan teknologi awal agar implementasi tidak berubah-ubah saat coding. Keputusan ini berlaku untuk fase MVP/demo production-like.

## 2. Ringkasan Keputusan

| Area | Keputusan |
| --- | --- |
| Repo model | Monorepo dengan 3 service utama |
| Auth | JWT middleware/edge concern, bukan `auth-service` |
| Go module | `go.work` dengan `go.mod` per service |
| Framework | `go-kratos` |
| PostgreSQL access order-service | `sqlc` + `pgx` / `pgxpool` |
| PostgreSQL access catalog-inventory-service | `sqlc` + `pgx` / `pgxpool` |
| PostgreSQL access payment-service | `sqlx` + `pgx` stdlib driver |
| Migration tool | `goose` |
| Kafka broker | Apache Kafka KRaft mode |
| Kafka client | `github.com/twmb/franz-go/pkg/kgo` |
| Redis client | `github.com/redis/go-redis` |
| Logger | `log/slog` |
| Tracing | OpenTelemetry |
| Local tracing backend | OpenTelemetry Collector + Jaeger |
| Config | Environment variables dengan typed config struct |
| Testing | Go `testing`, fake port, integration test dengan Docker Compose |

## 3. Repo Model

Gunakan monorepo:

```text
services/order-service
services/catalog-inventory-service
services/payment-service
```

Alasan:

- lebih mudah dijalankan sebagai portfolio project;
- contract, deployment, dan test dapat dikelola dalam satu repo;
- tetap menjaga service boundary melalui `go.mod` per service.

Tidak ada `auth-service` pada MVP/demo.

## 4. Auth

Auth tetap ada, tetapi tidak menjadi service ke-4.

Auth diimplementasikan sebagai:

```text
JWT middleware di REST adapter / API gateway / edge layer
```

Alasan:

- fokus utama project adalah Saga, Kafka, gRPC, outbox/inbox, dan observability;
- `auth-service` akan menambah scope user management yang belum dibutuhkan;
- auth tetap bisa diekstrak di masa depan jika dibutuhkan.

## 5. Go Module Strategy

Gunakan:

```text
go.work
services/order-service/go.mod
services/catalog-inventory-service/go.mod
services/payment-service/go.mod
```

Alasan:

- setiap service dapat build/test secara independen;
- development lokal tetap mudah karena `go.work`;
- cocok untuk monorepo microservices.

## 6. Framework: go-kratos

Gunakan `go-kratos` untuk:

- HTTP transport;
- gRPC transport;
- middleware;
- service bootstrap;
- config integration jika diperlukan.

Aturan Hexagonal:

```text
Kratos hanya boleh berada di adapter/bootstrap layer.
Domain dan application layer tidak boleh import Kratos.
```

## 7. PostgreSQL Access

Gunakan pendekatan berbeda per service untuk pembelajaran dan tradeoff yang jelas.

| Service | Data access | Alasan |
| --- | --- | --- |
| `order-service` | `sqlc` + `pgx` / `pgxpool` | Butuh type-safety untuk order lifecycle, saga state, outbox, dan inbox. |
| `catalog-inventory-service` | `sqlc` + `pgx` / `pgxpool` | Butuh SQL eksplisit dan aman untuk stock reservation, row locking, dan concurrency. |
| `payment-service` | `sqlx` + `pgx` stdlib driver | Query lebih sederhana, cocok untuk belajar manual mapping dan fleksibilitas SQL. |

Tidak menggunakan full ORM seperti GORM pada fase awal.

Alasan:

- query penting tetap eksplisit;
- lebih mudah menjelaskan SQL saat interview;
- `sqlc` memberi compile-time type safety untuk service yang lebih kritikal;
- `sqlx` memberi pengalaman manual mapping pada service yang risikonya lebih rendah;
- semua tetap berada di outbound adapter dan tidak bocor ke domain/application layer.

Aturan:

- `order-service` dan `catalog-inventory-service` menggunakan generated query dari `sqlc`;
- `payment-service` menggunakan `sqlx` dengan driver `pgx` stdlib;
- application layer bergantung pada repository port/interface;
- transaction runner dibungkus sebagai port agar use case tidak bergantung pada `pgx.Tx`, `sql.Tx`, atau `sqlx.Tx`.

Contoh driver `sqlx` untuk payment-service:

```go
import _ "github.com/jackc/pgx/v5/stdlib"

db, err := sqlx.Connect("pgx", databaseURL)
```

## 7.1 sqlc

Digunakan oleh:

```text
order-service
catalog-inventory-service
```

Struktur yang disarankan:

```text
internal/adapter/outbound/postgres/
  query/
    orders.sql
    outbox.sql
    inbox.sql
  sqlc/
    generated files
  repository.go
```

Aturan:

- SQL disimpan eksplisit di file `.sql`;
- query yang membutuhkan lock seperti `FOR UPDATE` harus ditulis jelas;
- generated code sqlc hanya digunakan di adapter outbound;
- domain/application tidak boleh import package sqlc generated.

## 7.2 sqlx

Digunakan oleh:

```text
payment-service
```

Struktur yang disarankan:

```text
internal/adapter/outbound/postgres/
  payment_repository_sqlx.go
  outbox_repository_sqlx.go
  inbox_repository_sqlx.go
  transaction_runner_sqlx.go
```

Aturan:

- gunakan named query atau explicit scan secara konsisten;
- mapping dari database row ke domain model dilakukan di repository adapter;
- domain/application tidak boleh import `sqlx`;
- jangan campur `sqlc` dan `sqlx` dalam service yang sama pada fase awal.

## 8. Migration: goose

Gunakan `goose` untuk migration per service.

Struktur:

```text
services/order-service/migrations
services/catalog-inventory-service/migrations
services/payment-service/migrations
```

## 9. Kafka Broker: Apache Kafka KRaft

Gunakan Apache Kafka dalam KRaft mode untuk local development.

Alasan:

- belajar Kafka asli;
- tidak perlu Zookeeper;
- lebih relevan dengan versi Kafka modern.

Local listener:

```text
kafka:9092       # container-to-container
localhost:29092 # host-to-kafka
```

## 10. Kafka Client: franz-go/kgo

Gunakan:

```text
github.com/twmb/franz-go/pkg/kgo
```

Alasan:

- pure Go Kafka client;
- fitur kuat untuk production-like project;
- mendukung consumer group, manual offset commit, idempotent/transactional producer, dan hook observability;
- cocok untuk portfolio distributed system.

Catatan penting:

```text
Kafka transaction tidak menggantikan outbox/inbox.
```

Outbox/inbox tetap wajib karena kita perlu atomicity antara local database transaction dan event publishing secara application-level.

## 11. Redis: go-redis

Gunakan:

```text
github.com/redis/go-redis
```

Redis hanya digunakan sebagai:

- cache;
- optional short-lived lock;
- fallback performance layer.

Redis bukan source of truth.

## 12. Logger: slog

Gunakan `log/slog` untuk structured logging.

Log wajib menyertakan field berikut jika tersedia:

```text
service
request_id
correlation_id
trace_id
span_id
order_id
event_id
event_type
```

## 13. OpenTelemetry Backend Lokal

Gunakan:

```text
OpenTelemetry Collector + Jaeger
```

Alasan:

- setup lokal sederhana;
- mudah menunjukkan distributed trace saat demo;
- cukup untuk kebutuhan portfolio.

## 14. Testing

Testing menggunakan:

- Go standard `testing`;
- fake port untuk unit test application layer;
- Docker Compose dependency untuk integration test;
- test script untuk E2E scenario.

Detail strategi test ada di:

```text
doc/testing/testing-strategy.md
```
