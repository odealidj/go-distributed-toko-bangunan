# ADR-007: Gunakan OpenTelemetry untuk Distributed Tracing

## Status

Accepted

## Konteks

Checkout flow melewati REST, gRPC, Kafka, PostgreSQL, Redis, dan beberapa service. Debugging tanpa distributed tracing akan sulit.

## Keputusan

Gunakan OpenTelemetry untuk distributed tracing di semua Go service.

Trace context harus dipropagasi melalui:

- REST headers.
- gRPC metadata.
- Kafka headers.

Backend local/demo:

```text
OpenTelemetry Collector -> Jaeger or Grafana Tempo
```

## Konsekuensi

- Checkout flow dapat diinspeksi end-to-end.
- Log harus menyertakan trace dan correlation field.
- Kafka producer/consumer adapter harus menangani trace header.
- Tracer exporter failure tidak boleh merusak core business flow dalam demo mode.
