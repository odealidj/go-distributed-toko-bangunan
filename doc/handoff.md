# Handoff Project

Dokumen ini adalah ringkasan operasional untuk melanjutkan pekerjaan di sesi Codex baru atau oleh agent lain. Gunakan dokumen ini bersama `README.md`, `doc/README.md`, dan `doc/implementation/implementation-phases.md`.

## 1. Tujuan Task

Membangun demo microservices toko bangunan dengan Go yang cukup kuat untuk portfolio/CV.

Target utama:

- monorepo dengan 3 service:
  - `order-service`;
  - `catalog-inventory-service`;
  - `payment-service`;
- Hexagonal Architecture per service;
- PostgreSQL sebagai durable source of truth;
- Redis sebagai cache non-kritikal;
- gRPC untuk sync internal;
- Kafka untuk async event;
- Saga orchestration untuk checkout;
- outbox/inbox pattern;
- idempotent Kafka consumer;
- OpenTelemetry tracing;
- test untuk failure scenario;
- CI pipeline;
- K6 smoke/load test;
- README portfolio.

## 2. Branch dan Commit Terakhir

Branch aktif terakhir:

```text
phase/10-kafka-dlq-retry
```

Commit terakhir:

```text
cek `git log -1 --oneline` pada branch aktif
```

Branch phase yang sudah dibuat:

```text
phase/01-kontrak-tooling-runtime
phase/02-catalog-inventory-core
phase/03-payment-core
phase/04-order-checkout-saga
phase/05-kafka-outbox-inbox
phase/06-observability
phase/07-testing-failure-scenario
phase/08-ci-performance-portfolio
phase/09-metrics-dashboard
phase/10-kafka-dlq-retry
```

## 3. File dan Area yang Sudah Diubah

Ringkasan area perubahan besar:

| Area | File/Folder |
| --- | --- |
| Service order | `services/order-service/` |
| Service catalog/inventory | `services/catalog-inventory-service/` |
| Service payment | `services/payment-service/` |
| Shared helper | `shared/` |
| Proto/gRPC contract | `proto/` |
| Docker Compose | `docker-compose.yml` |
| Docker build | `deployments/docker/service.Dockerfile` |
| PostgreSQL init | `deployments/docker/postgres/init.sql` |
| OpenTelemetry Collector | `deployments/otel/collector.yaml` |
| Make target | `Makefile` |
| E2E/helper script | `scripts/` |
| K6 test | `tests/performance/k6/` |
| CI | `.github/workflows/ci.yml` |
| Dokumentasi | `README.md`, `doc/` |

File penting yang paling sering perlu dibaca:

```text
README.md
Makefile
docker-compose.yml
doc/README.md
doc/implementation/implementation-phases.md
doc/implementation/technology-decisions.md
doc/implementation/repository-structure.md
doc/architecture/checkout-saga.md
doc/testing/testing-strategy.md
doc/testing/performance-testing-k6.md
doc/observability/tracing-and-idempotency.md
doc/demo/demo-script.md
```

## 4. Keputusan Penting

Keputusan arsitektur:

- Checkout memakai Saga orchestration.
- Orchestrator masih berada di `order-service`.
- Desain tetap disiapkan agar orchestration bisa diekstrak menjadi service sendiri di masa depan.
- Kafka dipakai untuk domain event async.
- gRPC dipakai untuk komunikasi sync internal.
- Outbox dipakai untuk publish event secara reliable setelah local transaction commit.
- Inbox dipakai untuk idempotent Kafka consumer.
- Kafka delivery diasumsikan at-least-once.
- Business correctness tidak bergantung pada Kafka exactly-once.
- Redis dipakai sebagai cache/fallback performance, bukan source of truth.
- PostgreSQL tetap menjadi durable source of truth untuk setiap service.
- `payment-service` memakai `sqlx` sebagai pembelajaran, sedangkan `order-service` dan `catalog-inventory-service` memakai `sqlc`.
- Logger memakai `slog`.
- Tracing memakai OpenTelemetry.
- Kafka client memakai `franz-go/kgo`.

Keputusan implementasi:

- Custom response JSON sudah distandardisasi melalui `shared/response`.
- REST response data memakai DTO eksplisit, bukan anonymous `map[string]any`.
- `request_id` dan `correlation_id` dipropagasi lewat HTTP/gRPC/Kafka.
- Kafka header membawa `traceparent`, `x-correlation-id`, `x-causation-id`, `x-event-id`, dan `x-event-type`.
- Kafka consumer memakai exponential backoff retry dan publish ke DLQ setelah retry limit.
- K6 test dijalankan via container image `grafana/k6`.
- Local default memakai Podman, tetapi target `Makefile` bisa diganti ke Docker dengan `COMPOSE="docker compose"` atau `CONTAINER=docker`.

## 5. Error Terakhir dan Statusnya

Tidak ada error terbuka terakhir saat handoff ini dibuat.

Error penting yang pernah ditemukan dan sudah diperbaiki:

- Duplicate Kafka event di inventory/payment sempat menyebabkan transaction PostgreSQL `aborted` karena insert ke `inbox_events` terkena unique violation.
  - Perbaikan: insert inbox memakai `ON CONFLICT (event_id) DO NOTHING`.
- Consumer inventory sempat berhenti saat menerima terminal order event untuk order yang tidak punya reservation lokal.
  - Perbaikan: event terminal tanpa reservation lokal diperlakukan sebagai no-op terkontrol.
- `test-e2e` sempat gagal `404 PRODUCT_NOT_FOUND` karena data seed tidak tersedia setelah integration test.
  - Perbaikan: `test-e2e` menjalankan seed inventory demo di awal.
- Script trace sempat salah payload request `POST /orders`.
  - Perbaikan: payload mengikuti DTO `customer` + `items`.
- Metrics stack sempat gagal saat memakai cAdvisor pada Podman rootless Ubuntu.
  - Perbaikan: diganti ke `node-exporter` agar stabil pada Podman dan Docker.

## 6. Langkah Berikutnya

Langkah paling masuk akal berikutnya:

1. Buat Pull Request dari branch phase terakhir ke `main`.
2. Pastikan GitHub Actions CI hijau.
3. Review ulang `README.md` dari sudut pandang recruiter/backend reviewer.
4. Jalankan demo manual sesuai `doc/demo/demo-script.md`.
5. Ambil screenshot Jaeger trace untuk portfolio.
6. Ambil screenshot Grafana dashboard dan Kafka UI topic/consumer lag untuk portfolio.
7. Jika ingin lanjut production-like:
   - perluas dashboard Grafana;
   - tambah GitHub Actions integration test dengan service container.
   - tambah business metrics untuk outbox/inbox/DLQ.

## 7. Command yang Harus Dijalankan

### Setup infra lokal

```bash
make infra-up
make kafka-topics
make up
make inventory-seed
```

Jika memakai Docker Compose:

```bash
make infra-up COMPOSE="docker compose"
make up COMPOSE="docker compose"
```

### Validasi cepat

```bash
GOCACHE=/tmp/go-build-cache make test-unit
GOCACHE=/tmp/go-build-cache make build-all
make proto-validate
```

### Integration dan E2E

```bash
make test-integration
make test-e2e
```

### Observability

```bash
make trace-verify
PAYMENT_MODE=FAILURE make trace-verify
```

Jaeger UI:

```text
http://localhost:16686
```

### K6

```bash
make perf-smoke
make perf-load
```

Jika memakai Docker runtime:

```bash
make perf-smoke CONTAINER=docker
```

### Metrics dashboard

```bash
make metrics-up
```

URL:

```text
Prometheus: http://localhost:9090
Grafana: http://localhost:3000
Node exporter: http://localhost:9100/metrics
```

### Docker Compose check

```bash
podman compose --profile app config
```

atau:

```bash
docker compose --profile app config
```

## 8. Acceptance Criteria yang Sudah Terpenuhi

Sudah terpenuhi:

- 3 service utama berjalan.
- Checkout success menghasilkan order `CONFIRMED`.
- Insufficient stock menghasilkan order `REJECTED`.
- Payment failure menghasilkan order `CANCELLED`.
- Compensation release stock berjalan.
- Outbox publish ke Kafka berjalan.
- Inbox idempotency berjalan.
- Duplicate Kafka event tidak menggandakan business effect.
- OpenTelemetry trace terlihat di Jaeger.
- `make test-e2e` menjalankan failure scenario utama.
- `make perf-smoke` menjalankan K6 smoke test.
- CI workflow sudah dibuat.
- README portfolio sudah diperkuat.
- `/metrics` service sudah tersedia.
- Metrics stack lokal dengan Prometheus/Grafana/node-exporter/exporter sudah tersedia.
- Kafka consumer retry/backoff dan DLQ dasar sudah tersedia.
- `make kafka-topics` membuat topic utama dan topic DLQ.

Perlu diverifikasi di GitHub:

- status CI benar-benar hijau setelah workflow berjalan di remote.

## 9. Known Gaps / Roadmap

Belum selesai dan masih layak jadi phase berikutnya:

- CI integration test dengan service container.
- load/stress test tuning setelah baseline nyata.
- perluasan dashboard Grafana dan dokumentasi screenshot hasil demo.

## 10. Cara Agent Baru Memulai

Urutan baca paling efisien:

1. `README.md`
2. `doc/handoff.md`
3. `doc/implementation/implementation-phases.md`
4. `doc/architecture/checkout-saga.md`
5. `doc/testing/testing-strategy.md`
6. `doc/observability/tracing-and-idempotency.md`
7. `Makefile`

Setelah itu jalankan:

```bash
git status --short --branch
make proto-validate
GOCACHE=/tmp/go-build-cache make test-unit
GOCACHE=/tmp/go-build-cache make build-all
```

Jika akan melanjutkan feature baru, buat branch baru dari branch phase terakhir atau dari `main` setelah PR phase terakhir digabung.
