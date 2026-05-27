# ADR-004: Gunakan gRPC untuk Komunikasi Sinkron Internal

## Status

Accepted

## Konteks

Beberapa internal operation membutuhkan response langsung dan strongly typed contract, seperti product validation, stock reservation, dan payment creation.

## Keputusan

Gunakan gRPC untuk komunikasi sinkron internal antar service.

Examples:

- `ValidateProducts`
- `ReserveStock`
- `CreatePayment`

## Konsekuensi

- Contract eksplisit melalui file `.proto`.
- Go service mendapatkan generated client/server yang type-safe.
- gRPC call harus memiliki timeout.
- gRPC command yang mengubah state harus idempotent.
- OpenTelemetry context harus dipropagasi melalui gRPC metadata.
