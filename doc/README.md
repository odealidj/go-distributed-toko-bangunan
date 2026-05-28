# Indeks Dokumentasi

Folder ini berisi spesifikasi produk dan teknis untuk demo microservices Mini Toko Bangunan.

## Dokumen Inti

- [Product Requirements Document](product/prd.md)
- [System Architecture](architecture/system-architecture.md)
- [Checkout Saga Design](architecture/checkout-saga.md)
- [Panduan Go Hexagonal Architecture](implementation/go-hexagonal-architecture.md)
- [Keputusan Teknologi](implementation/technology-decisions.md)
- [Struktur Repository](implementation/repository-structure.md)
- [Phase Implementasi](implementation/implementation-phases.md)
- [Strategi Redis Cache](cache/redis-cache.md)
- [Tracing, OpenTelemetry, dan Kafka Idempotency](observability/tracing-and-idempotency.md)
- [Metrics dan Resource Dashboard](observability/metrics-dashboard.md)
- [Roadmap Implementasi Production-Like](roadmap/production-like-implementation-roadmap.md)
- [Architecture Decision Records](adr/README.md)

## Spesifikasi Machine-Readable

- [Spesifikasi Service](specs/services.yaml)
- [Business Rules](specs/business-rules.yaml)
- [State Machines](specs/state-machines.yaml)
- [Kontrak OpenAPI REST](api/openapi.yaml)
- [Standar REST Response](api/response-standard.md)
- [Kontrak AsyncAPI Kafka](events/asyncapi.yaml)

## Kontrak gRPC

- [Inventory Proto](grpc/inventory.proto)
- [Payment Proto](grpc/payment.proto)
- [Order Proto](grpc/order.proto)

## Dokumen Pendukung

- [Kontrak Event](events/event-contracts.md)
- [Kafka Operational Design](events/kafka-operational-design.md)
- [Logical Data Model](database/logical-data-model.md)
- [Panduan Local Development](deployment/local-development.md)
- [Arsitektur Docker Compose](deployment/docker-compose-architecture.md)
- [Desain Auth](security/auth-design.md)
- [Strategi Testing](testing/testing-strategy.md)
- [Performance Testing dengan K6](testing/performance-testing-k6.md)
- [Demo Script](demo/demo-script.md)
- [Panduan Implementasi AI](prompts/implementation-guide.md)

## Cara Penggunaan

Gunakan dokumen ini sebagai source of truth saat mengimplementasikan sistem. Target implementasi saat ini adalah Go dengan Hexagonal Architecture, Kafka untuk async event, gRPC untuk komunikasi sinkron internal, PostgreSQL sebagai source of truth, dan Redis sebagai cache. Implementasi boleh memilih library Go tertentu, tetapi harus mempertahankan service boundary, contract, state machine, dan compensation behavior.
