# Phase Implementasi

## 1. Tujuan

Dokumen ini membagi pekerjaan coding Mini Toko Bangunan menjadi phase yang jelas, kecil, dan bisa direview satu per satu.

Setiap phase harus:

- punya branch sendiri;
- punya target teknis yang eksplisit;
- mengikuti dokumen arsitektur yang sudah ada;
- menghasilkan demo yang bisa dijelaskan;
- divalidasi dengan test atau command yang relevan;
- dicommit dengan pesan singkat bahasa Indonesia.

## 2. Branching Strategy

Branch utama:

```text
main
```

Aturan:

- `main` hanya berisi perubahan yang sudah stabil.
- Setiap phase dibuat dari `main` terbaru.
- Jika phase terlalu besar, pecah lagi menjadi branch `feature/*` dari branch phase.
- Nama branch menggunakan format kebab-case.
- Commit message menggunakan bahasa Indonesia singkat.

Format branch phase:

```text
phase/NN-nama-phase
```

Format branch feature:

```text
feature/nama-fungsi
```

Contoh:

```text
phase/02-catalog-inventory-core
feature/catalog-product-list
feature/order-checkout-saga
```

## 3. Definition of Done Umum

Sebuah phase dianggap selesai jika:

- code mengikuti Hexagonal Architecture;
- REST response mengikuti `doc/api/response-standard.md`;
- state transition mengikuti `doc/specs/state-machines.yaml`;
- event mengikuti `doc/events/event-contracts.md` dan `doc/events/asyncapi.yaml`;
- gRPC mengikuti kontrak di `proto/` dan `doc/grpc/`;
- request/correlation ID tidak hilang antar boundary;
- test minimal untuk behavior utama tersedia;
- command validasi dicatat di final summary;
- tidak ada artifact build yang ikut commit.

## 4. Phase 00 - Fondasi Repository dan Infrastruktur

Status: selesai di `main`

Commit referensi:

```text
814aa12 siapkan fondasi kode
```

Cakupan:

- monorepo Go dengan `go.work`;
- 3 service module;
- shared module;
- Docker Compose untuk PostgreSQL, Redis, Kafka, Kafka UI, OpenTelemetry Collector, dan Jaeger;
- migration awal;
- proto awal;
- health endpoint awal;
- Makefile dasar.

Validasi:

```bash
make test-all
make build-all
docker compose --profile app config
protoc --proto_path=proto --descriptor_set_out=/tmp/toko-bangunan-proto.pb proto/inventory/v1/inventory.proto proto/order/v1/order.proto proto/payment/v1/payment.proto
```

## 5. Phase 01 - Kontrak, Tooling, dan Runtime Dasar

Branch:

```text
phase/01-kontrak-tooling-runtime
```

Tujuan:

Membuat contract dan runtime dasar siap dipakai sebelum business logic utama masuk.

Cakupan:

- generate gRPC code dari `proto/`;
- rapikan package generated code agar tidak masuk domain/application;
- middleware request ID dan correlation ID untuk HTTP;
- response JSON sukses/error/pagination;
- config loader per service;
- graceful shutdown;
- Makefile untuk generate, migrate, seed, dan menjalankan service;
- validasi contract lokal.

Rujukan:

- `doc/api/response-standard.md`
- `doc/grpc/*.proto`
- `doc/implementation/repository-structure.md`
- `doc/prompts/implementation-guide.md`

Acceptance criteria:

- semua service dapat start lokal;
- `/healthz` dan `/readyz` mengembalikan response standar;
- missing correlation ID akan dibuat otomatis;
- generated gRPC code tidak diimport oleh domain;
- `make test-all` dan `make build-all` lolos.

## 6. Phase 02 - Catalog Inventory Core

Branch:

```text
phase/02-catalog-inventory-core
```

Tujuan:

Membuat service catalog-inventory dapat membaca produk dan mengelola stock reservation.

Cakupan:

- domain model product, inventory, stock reservation;
- sqlc query dan repository PostgreSQL;
- seed product toko bangunan;
- REST product list/detail dengan pagination;
- Redis cache-aside untuk product read;
- gRPC:
  - `ValidateProducts`;
  - `ReserveStock`;
  - `CommitStock`;
  - `ReleaseStock`;
- outbox/inbox table adapter dasar.

Rujukan:

- `doc/database/logical-data-model.md`
- `doc/cache/redis-cache.md`
- `doc/specs/business-rules.yaml`
- `doc/specs/state-machines.yaml`

Acceptance criteria:

- product list bisa dipanggil dari REST;
- pagination memakai response standar;
- stock reservation memakai database transaction dan lock yang eksplisit;
- Redis down diperlakukan sebagai cache miss;
- duplicate reservation dengan idempotency key yang sama tidak menggandakan stock hold.

## 7. Phase 03 - Payment Core

Branch:

```text
phase/03-payment-core
```

Tujuan:

Membuat payment-service siap dipakai oleh Saga dengan behavior sukses/gagal yang bisa didemo.

Cakupan:

- domain model payment dan payment attempt;
- repository PostgreSQL memakai `sqlx` + driver `pgx`;
- gRPC:
  - `CreatePayment`;
  - `CancelPayment`;
- demo payment gateway adapter;
- forced success/failure untuk skenario demo;
- outbox/inbox table adapter dasar.

Rujukan:

- `doc/implementation/technology-decisions.md`
- `doc/specs/business-rules.yaml`
- `doc/specs/state-machines.yaml`
- `doc/grpc/payment.proto`

Acceptance criteria:

- payment creation idempotent berdasarkan idempotency key;
- payment failure dapat dipaksa untuk demo compensation;
- payment cancel tidak error jika event duplicate;
- application layer tidak import `sqlx`.

## 8. Phase 04 - Order Core dan Checkout Saga

Branch:

```text
phase/04-order-checkout-saga
```

Tujuan:

Membuat end-to-end checkout synchronous path berjalan dengan Saga orchestration di dalam `order-service`.

Cakupan:

- order domain model;
- order item price snapshot;
- saga state model;
- repository order, saga, outbox, inbox memakai `sqlc`;
- REST create checkout;
- REST get order detail;
- orchestration package terpisah agar mudah diekstrak menjadi service sendiri nanti;
- gRPC client ke inventory dan payment;
- timeout dan retry terbatas.

Rujukan:

- `doc/architecture/checkout-saga.md`
- `doc/adr/002-keep-checkout-orchestrator-inside-order-service.md`
- `doc/adr/001-use-saga-orchestration.md`
- `doc/specs/state-machines.yaml`

Acceptance criteria:

- checkout success menghasilkan order confirmed;
- insufficient stock menghasilkan order rejected;
- payment failed setelah stock reserved menghasilkan compensation release stock;
- saga transition tersimpan dan bisa diaudit;
- orchestration tidak bergantung langsung ke REST handler.

## 9. Phase 05 - Kafka Outbox, Inbox, dan Idempotent Consumer

Branch:

```text
phase/05-kafka-outbox-inbox
```

Tujuan:

Mengubah event flow menjadi reliable async dengan Kafka, outbox, inbox, dan idempotent consumer.

Cakupan:

- franz-go/kgo producer adapter;
- outbox publisher worker per service;
- Kafka consumer adapter per service;
- inbox atau processed event table;
- manual offset commit setelah local transaction sukses;
- Kafka header propagation:
  - `traceparent`;
  - `correlation_id`;
  - `causation_id`;
  - `idempotency_key`;
- duplicate event handling.

Rujukan:

- `doc/events/kafka-operational-design.md`
- `doc/events/event-contracts.md`
- `doc/observability/tracing-and-idempotency.md`
- `doc/adr/003-use-outbox-inbox.md`

Acceptance criteria:

- event dari outbox berhasil publish ke Kafka;
- consumer tidak menjalankan mutation dua kali untuk event duplicate;
- offset tidak dicommit sebelum transaction lokal sukses;
- Kafka key untuk checkout memakai `order_id`;
- duplicate event tercatat di log atau metric.

## 10. Phase 06 - Observability Production-Like

Branch:

```text
phase/06-observability
```

Tujuan:

Membuat demo trace dan log distributed flow terlihat jelas.

Cakupan:

- OpenTelemetry setup per service;
- tracing HTTP, gRPC, Kafka, PostgreSQL, Redis;
- slog structured logger;
- log field:
  - `trace_id`;
  - `span_id`;
  - `correlation_id`;
  - `order_id`;
  - `service`;
- Jaeger local verification;
- helper script untuk membuka atau memverifikasi trace.

Rujukan:

- `doc/observability/tracing-and-idempotency.md`
- `doc/observability/metrics-dashboard.md`
- `doc/adr/007-use-opentelemetry.md`

Acceptance criteria:

- satu checkout success bisa dilihat dari REST sampai Kafka di Jaeger;
- compensation flow punya trace yang bisa diikuti;
- log bisa difilter berdasarkan `correlation_id`;
- propagation context tidak hilang di Kafka header.

## 11. Phase 07 - Testing Failure Scenario

Branch:

```text
phase/07-testing-failure-scenario
```

Tujuan:

Membuktikan behavior distributed transaction dan idempotency lewat test.

Cakupan:

- unit test domain state transition;
- use case test dengan fake port;
- integration test repository;
- E2E checkout success;
- E2E insufficient stock;
- E2E payment failed compensation;
- duplicate Kafka event test;
- Redis unavailable test.

Rujukan:

- `doc/testing/testing-strategy.md`
- `doc/demo/demo-script.md`
- `doc/roadmap/production-like-implementation-roadmap.md`

Acceptance criteria:

- failure scenario dapat dijalankan dari command line;
- duplicate event tidak menggandakan business effect;
- test result bisa dipakai untuk cerita portfolio;
- test tidak membutuhkan manual step selain infra lokal.

## 12. Phase 08 - CI, K6, Metrics, dan Portfolio README

Branch:

```text
phase/08-ci-performance-portfolio
```

Tujuan:

Membuat project siap ditampilkan di CV dan mudah direview recruiter/engineer.

Cakupan:

- GitHub Actions:
  - test;
  - build;
  - proto validation;
  - docker build;
- K6 smoke/load test;
- resource metrics stack jika sudah siap;
- README portfolio;
- demo script final;
- ADR index final.

Rujukan:

- `doc/testing/performance-testing-k6.md`
- `doc/observability/metrics-dashboard.md`
- `doc/roadmap/production-like-implementation-roadmap.md`

Acceptance criteria:

- CI hijau di GitHub;
- demo dapat dijalankan dari instruksi README;
- K6 smoke test bisa membuktikan endpoint utama sehat;
- README menjelaskan Saga, outbox/inbox, idempotency, tracing, dan tradeoff.

## 13. Urutan Eksekusi Praktis

Urutan kerja yang direkomendasikan:

1. `phase/01-kontrak-tooling-runtime`
2. `phase/02-catalog-inventory-core`
3. `phase/03-payment-core`
4. `phase/04-order-checkout-saga`
5. `phase/05-kafka-outbox-inbox`
6. `phase/06-observability`
7. `phase/07-testing-failure-scenario`
8. `phase/08-ci-performance-portfolio`

Alasan:

- contract dan tooling dibuat dulu agar perubahan berikutnya stabil;
- inventory dan payment dibuat sebelum order Saga karena Saga membutuhkan downstream;
- outbox/inbox dan Kafka dipasang setelah local transaction behavior jelas;
- observability dan testing diperkuat setelah flow utama benar.

