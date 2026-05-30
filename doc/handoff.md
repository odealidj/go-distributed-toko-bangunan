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
phase/13-business-completeness
```

Commit terakhir:

```text
cek `git log -1 --oneline` pada branch aktif
```

Commit sebelumnya yang relevan:

```text
26bc376 lengkapi flow bisnis
bac0c11 tambah panduan portfolio
88715a4 rapikan handoff ci
45e4571 stabilkan e2e kafka
5ff6881 update handoff ci
bbaee32 perkuat readiness ci
72999f8 rapikan workflow ci
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
phase/11-ci-integration
phase/12-portfolio-showcase
phase/13-business-completeness
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
doc/demo/portfolio-showcase.md
```

File yang terakhir berubah dan relevan untuk business completeness:

```text
services/order-service/internal/adapter/inbound/rest/orders.go
services/order-service/internal/adapter/inbound/kafka/payment_events_consumer.go
services/order-service/internal/adapter/outbound/postgres/order_repository.go
services/payment-service/internal/adapter/inbound/rest/payments.go
services/payment-service/internal/adapter/outbound/postgres/payment_repository_sqlx.go
services/payment-service/internal/adapter/outbound/postgres/outbox_repository.go
services/payment-service/internal/application/worker/outbox_publisher.go
scripts/test-e2e.sh
```

File yang terakhir berubah dan relevan untuk investigasi CI:

```text
.github/workflows/ci.yml
scripts/ci-integration.sh
scripts/test-e2e.sh
services/order-service/internal/adapter/inbound/rest/health.go
services/catalog-inventory-service/internal/adapter/inbound/rest/health.go
services/payment-service/internal/adapter/inbound/rest/health.go
README.md
doc/deployment/docker-compose-architecture.md
doc/deployment/local-development.md
doc/testing/testing-strategy.md
```

File yang terakhir berubah dan relevan untuk showcase portfolio:

```text
README.md
doc/demo/portfolio-showcase.md
doc/README.md
doc/implementation/implementation-phases.md
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
- CI integration stack dijalankan lewat `scripts/ci-integration.sh` dengan `docker compose`.
- CI dan E2E menunggu `/readyz`, bukan `/healthz`, agar startup gating benar-benar mencerminkan dependency inti.
- Local default memakai Podman, tetapi target `Makefile` bisa diganti ke Docker dengan `COMPOSE="docker compose"` atau `CONTAINER=docker`.

## 5. Error Terakhir dan Statusnya

Error/status terakhir saat handoff ini diperbarui:

- Commit `bbaee32 perkuat readiness ci` sudah memperbaiki gating startup lokal:
  - `scripts/ci-integration.sh` menunggu `/readyz`, bukan `/healthz`;
  - `scripts/test-e2e.sh` juga menunggu `/readyz` dan mengecek `payment-service`;
  - `readyz` di tiga service sekarang mengembalikan `503` jika dependency inti belum siap.
- Validasi lokal untuk commit tersebut lolos:
  - `bash -n scripts/ci-integration.sh scripts/test-e2e.sh`
  - `GOCACHE=/tmp/go-build-cache make test-unit`
  - `GOCACHE=/tmp/go-build-cache make build-all`
  - `make ci-integration COMPOSE="docker compose"`
- Mitigasi timing terbaru untuk runner CI:
  - `scripts/test-e2e.sh` tidak lagi memakai `sleep 2` untuk duplicate event Kafka.
  - Validasi duplicate event sekarang memakai polling bounded sampai inbox inventory/payment settle.
  - Fallback Redis pada E2E juga dipanggil lewat retry bounded.
- Status remote terbaru sudah hijau:
  - GitHub Actions run `26681168668` untuk commit `5ff6881` berstatus `completed/success` pada `2026-05-30`.
  - GitHub Actions run `26681209486` untuk commit `45e4571` berstatus `completed/success` pada `2026-05-30`.
  - Rangkaian kegagalan remote sebelumnya pada `26633670499` dan `26633113018` sudah tertutup.
- Phase terbaru yang sedang dikerjakan:
  - branch `phase/13-business-completeness`;
  - targetnya menutup gap `GET /orders`, `POST /orders/{id}/cancel`, manual payment resolution, dan `payment.events` consumer pada `order-service`.
  - implementasi utama sudah masuk pada commit `26bc376 lengkapi flow bisnis`.
- Validasi lokal untuk phase 13 yang sudah lolos:
  - `bash -n scripts/test-e2e.sh`
  - `GOCACHE=/tmp/go-build-cache make test-unit`
  - `GOCACHE=/tmp/go-build-cache make build-all`
- Catatan environment lokal terbaru:
  - re-run full stack seperti `make ci-integration COMPOSE="docker compose"` atau `make test-e2e` sempat terblokir konflik host port `6379`;
  - pemicunya container lain di mesin ini: `flashsale-redis`;
  - ini terlihat sebagai konflik environment lokal, bukan bukti regresi pada perubahan script atau code phase 13.

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

1. Push branch `phase/13-business-completeness`.
2. Bebaskan host port `6379`, lalu jalankan ulang `make ci-integration COMPOSE="docker compose"` atau `make test-e2e`.
3. Buat Pull Request dari branch phase terakhir ke `main`.
4. Review ulang `README.md` dari sudut pandang recruiter/backend reviewer.
5. Jalankan demo manual sesuai `doc/demo/demo-script.md` dan `doc/demo/portfolio-showcase.md`.
6. Ambil screenshot Jaeger trace untuk portfolio.
7. Ambil screenshot Grafana dashboard dan Kafka UI topic/consumer lag untuk portfolio.
8. Ambil screenshot GitHub Actions run terbaru yang hijau.

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
make ci-integration COMPOSE="docker compose"
```

Command tambahan untuk investigasi CI remote:

```bash
curl -fsSL "https://api.github.com/repos/odealidj/go-distributed-toko-bangunan/actions/runs?branch=phase/11-ci-integration&per_page=3" | jq
curl -fsSL "https://api.github.com/repos/odealidj/go-distributed-toko-bangunan/actions/runs/26681209486/jobs" | jq
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
- Panduan showcase portfolio sudah tersedia.
- `/metrics` service sudah tersedia.
- Metrics stack lokal dengan Prometheus/Grafana/node-exporter/exporter sudah tersedia.
- Kafka consumer retry/backoff dan DLQ dasar sudah tersedia.
- `make kafka-topics` membuat topic utama dan topic DLQ.
- CI workflow sudah menyiapkan job integration/e2e berbasis stack container.

Perlu diverifikasi di GitHub:

- status CI benar-benar hijau setelah workflow berjalan di remote.

## 9. Known Gaps / Roadmap

Belum selesai dan masih layak jadi phase berikutnya:

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
make ci-integration COMPOSE="docker compose"
```

Jika akan melanjutkan feature baru, buat branch baru dari branch phase terakhir atau dari `main` setelah PR phase terakhir digabung.
