# Panduan Showcase Portfolio

## 1. Tujuan

Dokumen ini membantu menampilkan project dengan bentuk yang kuat untuk recruiter, interviewer backend, atau reviewer distributed systems.

Target utamanya:

- pembaca paham sistem dalam 3-5 menit;
- pembaca melihat bahwa project ini lebih dari CRUD;
- artefak demo yang ditampilkan konsisten dengan implementasi dan test yang benar-benar jalan.

## 2. Artefak Minimum yang Harus Ada

Minimal siapkan 6 artefak berikut:

1. screenshot README bagian arsitektur dan nilai portfolio;
2. screenshot GitHub Actions run yang hijau;
3. screenshot Jaeger trace untuk checkout success;
4. screenshot Grafana dashboard `Toko Bangunan Overview`;
5. screenshot Kafka UI yang menunjukkan topic utama dan topic DLQ;
6. screenshot terminal saat `make test-e2e` atau `make ci-integration COMPOSE="docker compose"` berjalan sukses.

Jika hanya sempat menyiapkan 3 artefak, prioritaskan:

1. Jaeger trace;
2. GitHub Actions run;
3. terminal `make test-e2e`.

## 3. Persiapan Sebelum Capture

Jalankan stack lokal:

```bash
make infra-up
make kafka-topics
make up
make inventory-seed
make metrics-up
```

Validasi utama:

```bash
GOCACHE=/tmp/go-build-cache make test-unit
GOCACHE=/tmp/go-build-cache make build-all
make test-e2e
make trace-verify
```

Jika ingin memastikan CI-equivalent flow:

```bash
make ci-integration COMPOSE="docker compose"
```

Catatan:

- jika host port `6379` sedang dipakai proses lain, matikan dulu konflik lokal sebelum menjalankan compose stack ini;
- endpoint readiness yang dipakai untuk gating adalah `/readyz`, bukan `/healthz`.

## 4. Urutan Capture yang Disarankan

### A. README

Capture:

- judul project;
- diagram arsitektur;
- daftar nilai portfolio.

Pesan yang ingin disampaikan:

- project ini punya boundary service yang jelas;
- ada Saga, outbox/inbox, idempotency, tracing, dan failure scenario nyata.

### B. GitHub Actions

URL:

```text
https://github.com/odealidj/go-distributed-toko-bangunan/actions
```

Capture:

- workflow `ci`;
- run `26683645157` atau run terbaru yang hijau;
- job `integration-e2e` dan `docker-build`.

Pesan yang ingin disampaikan:

- project tidak hanya berjalan lokal;
- integration dan E2E diverifikasi di GitHub Actions dengan container stack nyata.

### C. Jaeger

URL:

```text
http://localhost:16686
```

Langkah:

1. jalankan `make trace-verify`;
2. buka trace terbaru untuk `order-service`;
3. tampilkan span checkout yang memanggil gRPC downstream dan menyentuh Kafka/database.

Capture yang dicari:

- trace parent-child yang jelas;
- span `CheckoutSaga.CreateCheckout`;
- span ke inventory/payment/PostgreSQL/Redis jika terlihat.

Pesan yang ingin disampaikan:

- ada observability lintas boundary, bukan log lokal satu service saja.

### D. Grafana

URL:

```text
http://localhost:3000
```

Credential default:

```text
admin / admin
```

Capture:

- dashboard `Toko Bangunan Overview`;
- panel request rate atau latency;
- panel process memory / CPU host jika tersedia.

Pesan yang ingin disampaikan:

- service punya metrics runtime yang bisa diamati saat demo/load test.

### E. Kafka UI

URL:

```text
http://localhost:8090
```

Capture:

- topic:
  - `order.events`
  - `inventory.events`
  - `payment.events`
  - `order.events.dlq`
  - `inventory.events.dlq`
  - `payment.events.dlq`
- consumer group jika terlihat.

Pesan yang ingin disampaikan:

- async event benar-benar dipakai;
- sistem juga menyiapkan jalur DLQ untuk kasus gagal.

### F. Terminal Test

Command:

```bash
make test-e2e
```

Atau:

```bash
make ci-integration COMPOSE="docker compose"
```

Capture:

- scenario success;
- insufficient stock;
- payment failed compensation;
- duplicate event idempotency;
- Redis unavailable fallback.

Pesan yang ingin disampaikan:

- distributed transaction dan failure path dibuktikan dengan test yang repeatable.

## 5. Narasi Demo Singkat

### Versi 30 detik

Project ini adalah demo microservices toko bangunan dengan tiga service Go. Checkout memakai Saga orchestration, komunikasi sinkron internal memakai gRPC, event async memakai Kafka dengan outbox/inbox, dan duplicate delivery ditahan oleh idempotent consumer. Saya juga menambahkan tracing, metrics, E2E failure scenarios, dan CI yang menyalakan container stack sungguhan.

### Versi 2 menit

Mulai dari `order-service`, checkout tidak langsung dianggap selesai. Service ini memvalidasi produk dan reserve stock ke inventory lewat gRPC, lalu membuat payment ke payment-service. Perubahan state bisnis disimpan bersama outbox event dalam transaction lokal. Event kemudian dipublish ke Kafka dan dikonsumsi service lain dengan inbox/idempotency. Jika payment gagal, order dibatalkan dan stock direlease lagi. Seluruh flow bisa dilihat lewat Jaeger, dan skenario gagal/duplicate event diverifikasi lewat `make test-e2e` serta GitHub Actions.

## 6. Talking Point Saat Interview

Topik yang aman dan kuat untuk dibahas:

- mengapa memilih Saga orchestration, bukan distributed transaction 2PC;
- kenapa business correctness tidak bergantung pada Kafka exactly-once;
- kenapa Redis hanya dijadikan cache non-kritikal;
- bagaimana desain sekarang memudahkan pemisahan orchestrator ke service sendiri;
- tradeoff biaya dan kompleksitas antara demo portfolio dan production penuh.

## 7. Checklist Sebelum Repo Dibagikan

- README sudah ringkas dan kuat di atas fold.
- branch phase terakhir dalam kondisi bersih.
- CI run terbaru hijau.
- `doc/handoff.md` sinkron dengan status terbaru.
- command `make infra-up`, `make kafka-topics`, `make up`, `make inventory-seed` masih sesuai.
- screenshot yang dipakai tidak menampilkan data sensitif.

## 8. Link Repo yang Paling Layak Ditunjukkan

- `README.md`
- `doc/demo/portfolio-showcase.md`
- `doc/architecture/checkout-saga.md`
- `doc/testing/testing-strategy.md`
- `.github/workflows/ci.yml`

## 9. Yang Tidak Perlu Dipaksakan

Untuk portfolio ini, Anda tidak perlu memaksakan:

- multi-node Kafka cluster;
- Kubernetes deployment penuh;
- service mesh;
- auth-service terpisah;
- benchmark skala besar yang tidak bisa dijelaskan tradeoff-nya.

Lebih baik menunjukkan sistem kecil tetapi koheren, bisa dijalankan, dan bisa dijelaskan sampai detail failure path.
