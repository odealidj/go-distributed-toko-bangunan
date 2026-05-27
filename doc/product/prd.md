# Product Requirements Document: Mini Toko Bangunan

## 1. Overview

Mini Toko Bangunan adalah aplikasi demo toko online untuk produk material bangunan. Produk ini dirancang untuk menunjukkan alur belanja sederhana sekaligus menjadi studi kasus microservices dengan REST, gRPC, Kafka, Saga orchestration, idempotency, dan distributed transaction compensation.

## 2. Tujuan

- Customer dapat melihat produk bangunan, membuat order, dan melihat status order.
- Admin/demo operator dapat melihat perubahan status order, stok, dan payment.
- Backend terdiri dari 3 service utama: `order-service`, `catalog-inventory-service`, dan `payment-service`.
- Sistem menunjukkan distributed transaction melalui checkout saga.
- Sistem menggunakan gRPC untuk komunikasi sinkron internal dan Kafka untuk event asinkron.
- Implementasi backend menggunakan Go dengan Hexagonal Architecture.
- Redis digunakan sebagai cache, bukan source of truth.
- Dokumentasi dapat menjadi acuan implementasi lintas teknologi.

## 3. Non-Goals

- Payment gateway production seperti Midtrans/Xendit.
- Integrasi kurir production.
- Multi-warehouse kompleks.
- Sistem promo, loyalty, invoice pajak, procurement supplier, dan accounting.
- Authentication production-grade. Untuk demo, auth dapat dibuat sederhana atau dimock.

## 4. Pengguna

### Customer

- Melihat katalog produk bangunan.
- Membuat order.
- Melihat status order.

### Admin/Demo Operator

- Mengelola produk dan stok awal.
- Memicu simulasi payment sukses/gagal.
- Mengamati alur Saga dan compensation.

## 5. Scope Service

### Order Service

- Frontend-facing service untuk checkout.
- Pemilik data order dan order item.
- Saga orchestrator untuk checkout.
- Mengubah status order berdasarkan hasil inventory dan payment.
- Publish event order ke Kafka.

### Catalog Inventory Service

- Pemilik data produk dan stok.
- Menyediakan detail produk.
- Melakukan reserve, commit, dan release stock.
- Publish event inventory ke Kafka.

### Payment Service

- Membuat payment record.
- Simulasi payment success/failure.
- Publish event payment ke Kafka.

## 6. Functional Requirements MVP

| ID | Requirement |
| --- | --- |
| FR-001 | Customer dapat melihat daftar produk aktif. |
| FR-002 | Customer dapat melihat detail produk termasuk unit, harga, dan stok tersedia. |
| FR-003 | Customer dapat membuat order dengan satu atau lebih item. |
| FR-004 | Sistem harus memvalidasi produk dan harga sebelum membuat order. |
| FR-005 | Sistem harus reserve stock sebelum membuat payment. |
| FR-006 | Jika stock tidak cukup, order harus berakhir dengan status `REJECTED`. |
| FR-007 | Jika payment sukses, order harus menjadi `CONFIRMED`. |
| FR-008 | Jika payment gagal setelah stock reserved, order harus menjadi `CANCELLED` dan stock harus direlease. |
| FR-009 | Consumer Kafka harus idempotent berdasarkan `event_id`. |
| FR-010 | Command gRPC yang mengubah state harus idempotent berdasarkan `idempotency_key`. |

## 7. Non-Functional Requirements

- Setiap service memiliki database sendiri.
- Tidak ada cross-service database join.
- Frontend berkomunikasi ke backend melalui REST API milik `order-service` dan read API katalog.
- Service internal berkomunikasi sinkron menggunakan gRPC.
- Domain event dipublish melalui Kafka.
- PostgreSQL menjadi source of truth untuk data durable.
- Redis digunakan untuk cache read model dan optional short-lived locks.
- Event penting harus dipublish menggunakan outbox pattern.
- Consumer harus menggunakan inbox/processed event table.
- Setiap request dan event harus membawa `correlation_id`.
- Error response REST harus konsisten.
- Service harus menyediakan health endpoint.

## 8. Asumsi Produk

- Produk toko bangunan memiliki satuan seperti `sak`, `batang`, `meter`, `dus`, `lembar`, `kg`, atau `m3`.
- Harga pada demo menggunakan snapshot saat order dibuat.
- Payment bersifat simulasi dan dapat dipaksa sukses/gagal untuk demo.
- Delivery tidak dihitung otomatis pada MVP; fokus demo adalah inventory-payment transaction.

## 9. Kriteria Sukses

- Demo checkout sukses memperlihatkan order confirmed, payment succeeded, dan stock committed.
- Demo stock insufficient memperlihatkan order rejected tanpa membuat payment.
- Demo payment failed memperlihatkan order cancelled dan stock released.
- Duplicate event tidak menyebabkan double commit/release.
- Dokumentasi cukup jelas untuk implementasi ulang di stack lain.
