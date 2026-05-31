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
b521958 tambah anotasi ci
bd3e03d perkuat retry ci
3e2f73f stabilkan ci compose
5322053 rapikan build lokal
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
.dockerignore
deployments/docker/service.Dockerfile
doc/deployment/local-development.md
doc/deployment/docker-compose-architecture.md
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
  - `go test -mod=readonly ./internal/adapter/outbound/postgres` pada `order-service`, `catalog-inventory-service`, dan `payment-service`
  - host-run E2E penuh dengan infra Compose + binary lokal:
    - `JAEGER_UI_PORT=16687 make infra-up`
    - `make order-migrate-compose inventory-migrate-compose payment-migrate-compose kafka-topics inventory-seed`
    - jalankan `./bin/order-service`, `./bin/catalog-inventory-service`, `./bin/payment-service`
    - `make test-e2e COMPOSE="podman compose"`
  - hasil terakhir: `semua scenario E2E berhasil`
- Validasi container penuh terbaru juga sudah lolos:
  - `JAEGER_UI_PORT=16687 make ci-integration COMPOSE="docker compose"`
  - perbaikan yang dibutuhkan:
    - `scripts/ci-integration.sh` kini menunggu PostgreSQL siap menerima koneksi dengan `pg_isready` sebelum migration;
    - ini menutup race saat startup yang sebelumnya memicu error `database system is shutting down`.
- Status remote terbaru setelah commit `3e2f73f stabilkan ci compose`:
  - GitHub Actions run `26683354945` pada branch `phase/13-business-completeness` gagal di job `integration-e2e`;
  - detail publik yang bisa diambil tanpa login masih terbatas pada `Process completed with exit code 2`;
  - artifact `ci-compose-logs` berhasil ter-upload, tetapi unduhan artifact GitHub membutuhkan autentikasi.
- Mitigasi terbaru yang sedang diterapkan:
  - `scripts/test-e2e.sh` kini memakai retry window yang lebih longgar untuk duplicate event, manual payment settle, dan Redis fallback;
  - saat settle gagal, script sekarang membuang snapshot state order/payment/inventory dan `compose ps` ke stderr agar log CI lebih informatif;
  - `scripts/ci-integration.sh` juga menambah window tunggu untuk `/readyz` dan `pg_isready`.
  - `scripts/ci-integration.sh` dan `scripts/test-e2e.sh` sekarang juga mengeluarkan GitHub Actions annotation (`::error::`) yang menyebut stage/scenario saat gagal, agar root cause bisa dilihat lewat Check Run API tanpa harus membuka log privat.
- Status remote terbaru saat ini:
  - GitHub Actions run `26683514775` untuk commit `bd3e03d` masih gagal di job `integration-e2e`;
  - follow-up observability/annotation ditambahkan pada commit `b521958 tambah anotasi ci`;
  - GitHub Actions run `26683645157` untuk commit `b521958` berstatus `completed/success` pada `2026-05-30`;
  - GitHub Actions run `26710619771` untuk commit `bfb6187` berstatus `completed/success` pada `2026-05-31`;
  - artinya phase 13 kini sudah lolos validasi lokal, validasi remote bisnis, dan status branch terbaru juga hijau.
- Catatan environment lokal terbaru:
  - konflik terbaru bukan lagi di Redis, tetapi di Jaeger host port `16686` karena stack lain aktif di mesin ini;
  - solusi yang sudah diterapkan: host port Compose kini bisa dioverride, misalnya `JAEGER_UI_PORT=16687`;
  - build image via `compose --build` di Podman tetap relatif lambat untuk validasi lokal;
  - mitigasi yang sudah diterapkan:
    - tambah `.dockerignore`;
    - rapikan cache layer pada `deployments/docker/service.Dockerfile`;
    - gunakan host-run service binary untuk validasi end-to-end lokal jika tidak butuh image build.

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

1. Review PR branch `phase/13-business-completeness`.
2. Lanjut ke artefak portfolio: screenshot GitHub Actions hijau, Jaeger, Grafana, dan Kafka UI.
3. Jika ingin lanjut coding feature baru, buat branch phase berikutnya dari branch ini atau dari `main` setelah PR digabung.

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

Sudah terverifikasi di GitHub:

- status CI branch `phase/13-business-completeness` hijau pada run `26683645157`, dan status branch terbaru juga hijau pada run `26710619771`.

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
