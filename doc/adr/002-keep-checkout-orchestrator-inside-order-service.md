# ADR-002: Pertahankan Checkout Orchestrator di Dalam Order Service untuk MVP

## Status

Accepted

## Konteks

Checkout lifecycle berpusat pada order. Dedicated orchestrator service akan menambah kompleksitas operasional sebelum workflow cukup besar untuk membenarkannya.

## Keputusan

Untuk MVP/demo, implementasikan checkout orchestration sebagai application module terpisah di dalam `order-service`.

Suggested package:

```text
order-service/internal/application/saga
```

## Konsekuensi

- Lebih sedikit service untuk dideploy dan didebug.
- Order ownership tetap jelas.
- Saga logic tidak boleh ditempatkan di REST handler atau repository.
- Saga module harus bergantung pada port/interface agar nantinya bisa diekstrak.
- Future extraction path tetap tersedia ketika checkout berkembang ke shipment, invoice, procurement, atau fulfillment workflow.
