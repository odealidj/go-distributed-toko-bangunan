# Roadmap Implementasi Production-Like

## 1. Tujuan

Roadmap ini mendefinisikan pekerjaan implementasi berikutnya agar demo Mini Toko Bangunan lebih mendekati sistem microservices production nyata dan cukup kuat untuk portfolio/CV.

Tujuannya bukan membuat sistem enterprise-heavy. Tujuannya adalah membuat distributed system kecil yang selesai, mudah dijelaskan, dapat dijalankan, observable, dan resilient terhadap failure case yang realistis.

## 2. Level Prioritas

| Priority | Arti |
| --- | --- |
| P0 | Wajib untuk demo portfolio end-to-end yang kredibel. |
| P1 | Peningkatan production-like yang kuat. |
| P2 | Nice to have setelah core demo stabil. |

## 3. Roadmap Implementasi

### 1. Security dan Auth

Priority: P1

Cakupan:

- JWT authentication.
- Roles: `CUSTOMER`, `ADMIN`.
- REST auth middleware.
- RBAC for admin/demo endpoints.
- Password hashing if user registration/login is implemented.
- Rate limiting for public endpoints.

Pendekatan yang direkomendasikan:

- Jaga auth tetap sederhana untuk demo.
- Jangan jadikan auth sebagai microservice terpisah dulu.
- Implementasikan auth middleware di API edge atau service REST adapter.

Kriteria penerimaan:

- Public product endpoint dapat diakses tanpa login.
- Order creation membutuhkan customer identity atau demo customer context.
- Admin/demo mutation endpoint membutuhkan `ADMIN`.
- JWT invalid mengembalikan standardized `401` error response.
- Role yang forbidden mengembalikan standardized `403` error response.

### 2. API Gateway / Edge Layer

Priority: P1

Cakupan:

- Add Nginx, Caddy, or a small Go gateway.
- Centralize CORS.
- Inject `X-Request-Id` and `X-Correlation-Id` when missing.
- Route frontend requests to services.
- Optional centralized auth/rate limit.

Pendekatan yang direkomendasikan:

- Untuk local dan cheap cloud deployment, gunakan Caddy atau Nginx.
- Jangan taruh business logic di gateway.

Kriteria penerimaan:

- Frontend/API client dapat memanggil satu base URL.
- Gateway meneruskan trace/correlation header.
- Gateway memiliki route yang terdokumentasi.
- Config gateway masuk ke Docker Compose.

### 3. Retry, Timeout, dan Circuit Breaker

Priority: P0

Cakupan:

- gRPC timeout per internal call.
- Retry only retryable errors.
- Backoff strategy.
- Circuit breaker for unstable downstream services.
- Same `idempotency_key` reused across retries.

Default timeout yang direkomendasikan:

| Call | Timeout | Retry |
| --- | --- | --- |
| `ValidateProducts` | 1s | 1 retry |
| `ReserveStock` | 2s | 2 retries |
| `CreatePayment` | 2s | 2 retries |
| `CommitStock` / `ReleaseStock` | async via Kafka | consumer retry |

Kriteria penerimaan:

- Downstream gRPC timeout tidak membuat caller hang tanpa batas.
- Retry tidak membuat duplicate stock reservation atau payment.
- Circuit breaker terbuka setelah failure berulang.
- Timeout dan retry behavior dapat dikonfigurasi.

### 4. Dead Letter Queue dan Replay

Priority: P1

Cakupan:

- DLQ topics:
  - `order.events.dlq`
  - `inventory.events.dlq`
  - `payment.events.dlq`
- Store failure reason and original event metadata.
- Provide manual replay command or script.

Pendekatan yang direkomendasikan:

- Mulai dengan DLQ publisher sederhana di Kafka consumer adapter.
- Tambahkan replay CLI nanti.

Kriteria penerimaan:

- Poison event diretry dengan backoff.
- Setelah max retry, event dipublish ke DLQ.
- DLQ event menyertakan original payload, headers, error reason, dan failed service.
- Replay menggunakan `event_id` yang sama, sehingga idempotency tetap melindungi state.

### 5. Database Migration dan Seed Strategy

Priority: P0

Cakupan:

- Migration folder per service.
- Seed data for demo products.
- Repeatable local database setup.
- Optional rollback scripts.

Struktur yang direkomendasikan:

```text
services/order-service/migrations
services/catalog-inventory-service/migrations
services/payment-service/migrations
services/catalog-inventory-service/seeds
```

Kriteria penerimaan:

- `make migrate` menjalankan migration untuk semua service.
- `make seed` memasukkan demo catalog dan stock.
- Database dapat direset dan direseed secara lokal.
- File migration dicommit dan deterministic.

### 6. CI Pipeline

Priority: P0

Cakupan:

- Run Go tests.
- Run linter.
- Validate proto contracts.
- Validate OpenAPI/AsyncAPI YAML.
- Build Docker images.

Job GitHub Actions yang direkomendasikan:

```text
lint
test
contract-validate
docker-build
```

Kriteria penerimaan:

- Pull request gagal jika test gagal.
- Pull request gagal jika file proto tidak compile.
- Pull request gagal jika YAML spec invalid.
- Docker image berhasil dibuild.

### 7. Testing Coverage yang Serius

Priority: P0

Cakupan:

- Domain unit tests.
- Application use case tests with fake ports.
- Repository integration tests.
- Kafka consumer idempotency tests.
- End-to-end checkout tests.

Test scenario yang wajib:

- Checkout success.
- Checkout rejected because stock is insufficient.
- Payment gagal setelah stock reserved.
- Duplicate `PaymentFailed` does not release stock twice.
- Duplicate `OrderConfirmed` does not commit stock twice.
- Redis unavailable does not break checkout correctness.

Kriteria penerimaan:

- Saga state transition dicover oleh test.
- Outbox/inbox behavior dicover oleh test.
- E2E demo scenario dapat dijalankan dari command line.

### 8. Observability Demo

Priority: P0

Cakupan:

- OpenTelemetry traces.
- Structured logs.
- Basic metrics.
- Jaeger or Tempo for local trace viewing.
- Resource metrics untuk CPU/RAM/container.
- Dashboard awal untuk Kafka lag, PostgreSQL connection, dan outbox pending.

Trace demo:

- Checkout success.
- Payment failed compensation.
- Duplicate event idempotency.

Kriteria penerimaan:

- Satu checkout flow dapat diikuti melalui span REST, gRPC, Kafka, PostgreSQL, dan Redis.
- Log menyertakan `trace_id`, `span_id`, `correlation_id`, dan `order_id`.
- Metric duplicate event Kafka terlihat atau dilog.
- CPU/RAM service terlihat saat demo.
- Kafka consumer lag dan outbox pending dapat dipantau.

### 8.1 Performance Testing dan Resource Metrics

Priority: P1

Cakupan:

- K6 smoke/load/stress/spike test.
- Custom metric checkout terminal duration.
- Prometheus dan Grafana untuk metrics dashboard.
- cAdvisor untuk container CPU/RAM/network/block I/O.
- postgres-exporter untuk PostgreSQL metrics.
- kafka-exporter untuk Kafka consumer lag/topic metrics.
- redis-exporter untuk Redis metrics jika cache sudah aktif.
- Business metrics untuk checkout, Saga, outbox, inbox, stock reservation, dan payment.

Dokumen:

```text
doc/testing/performance-testing-k6.md
doc/observability/metrics-dashboard.md
```

Kriteria penerimaan:

- `make perf-smoke` dapat menjalankan K6 smoke test.
- `make perf-load` menghasilkan summary latency, error rate, dan checkout terminal duration.
- Grafana menampilkan CPU/RAM container.
- Grafana menampilkan Kafka consumer lag.
- Grafana menampilkan PostgreSQL connection/query metrics.
- Grafana menampilkan outbox pending dan inbox duplicate metrics.
- Setelah load test, outbox pending dan Kafka lag dapat turun kembali.
- Tidak ada stock corruption setelah concurrent checkout.

### 9. Graceful Shutdown

Priority: P1

Cakupan:

- Stop HTTP server gracefully.
- Stop gRPC server gracefully.
- Stop Kafka consumers after current message handling.
- Stop outbox workers.
- Flush OpenTelemetry exporter.
- Close DB and Redis connections.

Kriteria penerimaan:

- Service menangani `SIGINT` dan `SIGTERM`.
- In-flight Kafka message tidak terputus di tengah local transaction.
- Tracer diflush sebelum process exit.

### 10. Configuration dan Secrets

Priority: P0

Cakupan:

- Typed config from environment variables.
- `.env.example`.
- No real secrets committed.
- Separate local/demo/prod-like config.

Env var yang wajib:

```text
DATABASE_URL
REDIS_ADDR
KAFKA_BROKERS
JWT_SECRET
OTEL_EXPORTER_OTLP_ENDPOINT
```

Kriteria penerimaan:

- Service fail fast jika required config hilang.
- `.env.example` mendokumentasikan semua variable yang wajib.
- Secret tidak disimpan di docs atau committed config.

### 11. Architecture Decision Records

Priority: P1

Cakupan:

- Dokumentasikan technical decision dan tradeoff penting.
- Jaga setiap ADR tetap pendek dan fokus.

ADR awal:

- Gunakan Saga orchestration.
- Pertahankan orchestrator di dalam `order-service` untuk MVP.
- Gunakan outbox/inbox untuk event reliability dan idempotency.
- Gunakan gRPC untuk komunikasi sinkron internal.
- Gunakan Kafka untuk domain event.
- Gunakan Redis hanya sebagai cache.
- Gunakan OpenTelemetry untuk distributed tracing.

Kriteria penerimaan:

- ADR ditautkan dari documentation index.
- Setiap ADR menyertakan context, decision, consequences, dan status.

### 12. One Command Local Demo

Priority: P0

Cakupan:

- Makefile or task runner.
- Docker Compose.
- Demo scripts for success/failure/idempotency scenarios.

Command yang direkomendasikan:

```text
make up
make down
make infra-up
make infra-down
make migrate
make seed
make order-run
make inventory-run
make payment-run
make order-test
make inventory-test
make payment-test
make test
make demo-success
make demo-payment-failed
make demo-insufficient-stock
make demo-duplicate-event
```

Kriteria penerimaan:

- Evaluator baru dapat menjalankan sistem dari instruksi README.
- Developer dapat menjalankan infrastructure saja dengan `make infra-up`.
- Setiap service dapat dijalankan dan dites lewat target Makefile masing-masing.
- Demo command mencetak expected final state.
- Reset command mengembalikan local environment ke clean state.

### 13. Cheap Cloud Deployment

Priority: P1

Cakupan:

- Single VPS deployment guide.
- Docker Compose production-like file.
- Caddy/Nginx reverse proxy.
- PostgreSQL backup script.
- Basic resource estimation.

Topologi low-cost yang direkomendasikan:

```text
1 VPS
Docker Compose
3 Go services
1 PostgreSQL instance with 3 databases
1 Redis instance
1 Kafka or Redpanda single-node instance
1 OpenTelemetry Collector
1 Jaeger/Tempo instance for demo
```

Kriteria penerimaan:

- Deployment guide menjelaskan resource yang dibutuhkan.
- Backup dan restore command terdokumentasi.
- Service restart otomatis setelah VPS reboot.
- Public endpoint dilindungi TLS jika diekspos.

### 14. Portfolio README

Priority: P0

Cakupan:

- Root README optimized for recruiters/interviewers.
- Architecture diagram.
- Tech stack.
- How to run.
- Demo scenarios.
- Distributed transaction explanation.
- Observability screenshots or instructions.
- Tradeoffs and future improvements.

Section README yang direkomendasikan:

```text
Problem
Architecture
Tech Stack
Service Responsibilities
Distributed Transaction Flow
Failure Scenarios
How to Run Locally
Demo Commands
Testing
Observability
Tradeoffs
Future Improvements
```

Kriteria penerimaan:

- Pembaca memahami sistem dalam 5 menit.
- Pembaca dapat menjalankan local demo tanpa pengetahuan privat.
- README menonjolkan mengapa project ini lebih dari CRUD.

## 4. Fase Implementasi yang Disarankan

### Phase 1: Foundation

Priority: P0

- Service skeletons in Go.
- Hexagonal package structure.
- PostgreSQL migrations.
- Docker Compose.
- OpenAPI/proto/event contracts.
- Basic Makefile.

### Phase 2: Core Business Flow

Priority: P0

- Catalog product reads.
- Inventory stock reservation.
- Order creation.
- Payment simulation.
- Checkout Saga.
- Outbox/inbox.
- Kafka event flow.

### Phase 3: Production-Like Reliability

Priority: P0/P1

- Timeout/retry.
- Idempotency tests.
- DLQ.
- Redis cache.
- Graceful shutdown.
- Config validation.

### Phase 4: Observability dan Demo Polish

Priority: P0/P1

- OpenTelemetry traces.
- Structured logs.
- Metrics.
- Demo scripts.
- Portfolio README.
- ADRs.
- CI pipeline.

### Phase 5: Cheap Cloud Deployment

Priority: P1

- VPS deployment guide.
- Reverse proxy.
- TLS.
- Backups.
- Resource estimation.

## 5. Yang Belum Perlu Ditambahkan

Hindari menambahkan hal berikut sampai core demo berjalan:

- Kubernetes.
- Service mesh.
- Multi-region deployment.
- Sharding.
- Complex auth service.
- Payment gateway integration.
- Multi-warehouse optimization.
- NoSQL rewrite.

Topik ini boleh disebut sebagai future improvement, tetapi tidak boleh mengalihkan fokus dari penyelesaian distributed transaction demo.
