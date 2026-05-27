# Performance Testing dengan K6

## 1. Tujuan

Dokumen ini mendefinisikan strategi performance test menggunakan K6 untuk demo Mini Toko Bangunan.

K6 digunakan untuk mengukur performa dari sisi client/API:

- latency HTTP;
- throughput;
- error rate;
- durasi checkout sampai terminal status;
- behavior saat traffic naik.

K6 tidak menggantikan observability internal seperti Prometheus, Grafana, Kafka lag, PostgreSQL metrics, dan resource metrics.

## 2. Kapan Dijalankan

Performance test dijalankan setelah:

1. E2E checkout success berjalan.
2. Payment failed compensation berjalan.
3. Outbox/inbox dan idempotent Kafka consumer berjalan.
4. OpenTelemetry trace dasar terlihat.
5. Seed data khusus load test tersedia.

Jangan jadikan K6 sebagai prioritas sebelum alur bisnis end-to-end stabil.

## 3. Scope Endpoint

Endpoint yang dites:

```text
GET /products
GET /products/{product_id}
POST /orders
GET /orders/{order_id}
```

Scenario checkout harus melakukan polling status order sampai terminal state.

Terminal state:

```text
CONFIRMED
CANCELLED
REJECTED
```

## 4. Struktur Folder

```text
tests/performance/k6/
├── smoke.js
├── load.js
├── stress.js
├── spike.js
├── soak.js
├── scenarios/
│   ├── browse-products.js
│   ├── checkout-success.js
│   ├── checkout-payment-failed.js
│   └── checkout-mixed.js
└── lib/
    ├── client.js
    ├── auth.js
    ├── polling.js
    └── metrics.js
```

## 5. Jenis Test

### Smoke Test

Tujuan:

- memastikan environment berjalan;
- memastikan endpoint utama merespons;
- memastikan script K6 valid.

Contoh target:

```text
VU: 1-5
Durasi: 1-3 menit
```

### Load Test

Tujuan:

- mengukur performa pada beban normal.

Contoh target:

```text
VU: 20-100
Durasi: 5-15 menit
```

### Stress Test

Tujuan:

- mencari batas sistem;
- melihat kapan latency/error/consumer lag mulai naik tajam.

Contoh target:

```text
VU naik bertahap: 50 -> 100 -> 200 -> 400
```

### Spike Test

Tujuan:

- melihat behavior saat traffic naik tiba-tiba.

Contoh target:

```text
VU: 10 -> 300 dalam waktu singkat -> 10
```

### Soak Test

Tujuan:

- melihat memory leak, goroutine leak, outbox backlog, dan resource drift.

Contoh target:

```text
VU: 50
Durasi: 30-120 menit
```

Untuk portfolio, soak test bersifat optional.

## 6. Checkout Flow di K6

Flow checkout tidak cukup hanya memvalidasi HTTP `201`.

Script harus:

```text
1. POST /orders
2. Ambil order_id
3. Poll GET /orders/{order_id}
4. Tunggu sampai terminal status:
   - CONFIRMED
   - CANCELLED
   - REJECTED
5. Catat durasi dari order dibuat sampai terminal status.
```

Timeout polling:

```text
default: 30 detik
```

Jika order belum terminal setelah timeout, hitung sebagai failure untuk performance scenario.

## 7. Custom Metrics K6

Metric K6 custom yang direkomendasikan:

```text
checkout_terminal_duration
checkout_confirmed_total
checkout_cancelled_total
checkout_rejected_total
checkout_timeout_total
checkout_success_rate
```

Metric bawaan K6 yang wajib diperhatikan:

```text
http_req_duration
http_req_failed
http_reqs
iterations
vus
checks
```

## 8. Threshold Awal

Threshold awal untuk local demo:

```text
http_req_failed < 1%
p95 GET /products < 500ms
p95 POST /orders < 1000ms
p95 checkout_terminal_duration < 10000ms
checkout_timeout_total == 0
```

Threshold ini dapat disesuaikan setelah baseline pertama.

## 9. Seed Data Load Test

Load test membutuhkan product khusus agar stock tidak cepat habis.

Contoh:

```text
prod_load_semen
name: Semen Load Test 50kg
stock: 1000000
price: 68000
unit: sak
```

Untuk scenario insufficient stock, gunakan product khusus:

```text
prod_load_low_stock
stock: 5
```

Jangan gunakan stock demo biasa untuk load test.

## 10. Command Makefile

Target yang direkomendasikan:

```text
make perf-smoke
make perf-load
make perf-stress
make perf-spike
make perf-soak
```

Contoh env:

```text
K6_BASE_URL=http://localhost:8080
K6_CUSTOMER_TOKEN=<jwt>
K6_ADMIN_TOKEN=<jwt>
```

## 11. Output

Untuk awal:

```text
k6 summary di terminal
```

Untuk versi lengkap:

```text
K6 -> Prometheus remote write -> Grafana dashboard
```

## 12. Hal yang Harus Dimonitor Saat K6

Saat K6 berjalan, pantau:

- CPU/RAM service;
- Kafka consumer lag;
- outbox pending events;
- PostgreSQL active connections;
- PostgreSQL slow query;
- Redis hit/miss;
- service error logs;
- Saga duration;
- checkout final status distribution.

Detail resource metrics ada di:

```text
doc/observability/metrics-dashboard.md
```

## 13. Kriteria Sukses

Performance test dianggap siap untuk portfolio jika:

- smoke test dapat dijalankan dengan satu command;
- load test menghasilkan summary yang jelas;
- final status checkout dapat diverifikasi;
- Grafana/metrics menunjukkan resource bottleneck utama;
- tidak ada stock corruption setelah concurrent checkout;
- outbox pending naik sementara lalu turun kembali;
- Kafka consumer lag tidak terus meningkat tanpa recovery.

