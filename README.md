# Go Distributed Toko Bangunan

Demo microservices untuk mini toko bangunan menggunakan Go, Hexagonal Architecture, gRPC, Kafka, PostgreSQL, Redis, OpenTelemetry, dan Saga orchestration.

Dokumentasi utama tersedia di [doc/README.md](doc/README.md).
Urutan coding dan branch per phase tersedia di [doc/implementation/implementation-phases.md](doc/implementation/implementation-phases.md).

## Service

- `order-service`
- `catalog-inventory-service`
- `payment-service`

## Quick Start

Local development default menggunakan Podman Compose.

```bash
make infra-up
make order-run
make inventory-run
make payment-run
```

Jika menggunakan Docker Compose:

```bash
make infra-up COMPOSE="docker compose"
```

## Dokumentasi

- [Keputusan Teknologi](doc/implementation/technology-decisions.md)
- [Struktur Repository](doc/implementation/repository-structure.md)
- [Arsitektur Docker Compose](doc/deployment/docker-compose-architecture.md)
- [Checkout Saga](doc/architecture/checkout-saga.md)
- [Kafka Operational Design](doc/events/kafka-operational-design.md)
