# Standar REST Response

## 1. Tujuan

Semua REST API yang menghadap frontend harus mengembalikan JSON envelope yang konsisten untuk success response dan error response.

Standar ini berlaku untuk:

- `order-service` REST APIs.
- `catalog-inventory-service` REST catalog APIs.
- Response API gateway di masa mendatang.

## 2. Success Response: Single Object

```json
{
  "success": true,
  "data": {
    "id": "ord_123",
    "status": "CONFIRMED"
  },
  "meta": {
    "request_id": "req_123",
    "correlation_id": "corr_123",
    "timestamp": "2026-05-27T10:00:00Z"
  }
}
```

Aturan:

- `success` harus bernilai `true`.
- `data` berisi response payload.
- `meta` wajib ada.
- `error` tidak boleh ada.
- Payload di `data` harus berasal dari DTO/response struct eksplisit pada adapter REST.
- Hindari `map[string]any` untuk response domain atau response operasional seperti health/readiness, kecuali payload benar-benar dinamis dan alasan teknisnya terdokumentasi.

Contoh DTO:

```go
type PaymentResponse struct {
	ID             string `json:"id"`
	OrderID        string `json:"order_id"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	PaymentMode    string `json:"payment_mode"`
	IdempotencyKey string `json:"idempotency_key"`
}
```

## 3. Success Response: List Dengan Pagination

```json
{
  "success": true,
  "data": [
    {
      "id": "prod_semen_50kg",
      "name": "Semen Tiga Roda 50kg"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total_items": 120,
    "total_pages": 6,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "req_123",
    "correlation_id": "corr_123",
    "timestamp": "2026-05-27T10:00:00Z"
  }
}
```

Aturan:

- List endpoint harus mengembalikan array di `data`.
- Empty list endpoint harus mengembalikan `data: []`.
- Paginated list endpoint harus menyertakan `pagination`.
- `pagination.total_items` hanya boleh dihilangkan jika implementasi secara eksplisit menggunakan cursor pagination di masa mendatang.

## 4. Error Response

```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Stock is not sufficient for one or more products.",
    "details": {
      "product_id": "prod_pasir_m3",
      "requested_qty": 99,
      "available_qty": 5
    }
  },
  "meta": {
    "request_id": "req_123",
    "correlation_id": "corr_123",
    "timestamp": "2026-05-27T10:00:00Z"
  }
}
```

Aturan:

- `success` harus bernilai `false`.
- `error.code` harus stabil dan machine-readable.
- `error.message` harus human-readable.
- `details` boleh berisi informasi debugging kontekstual yang aman untuk client.
- Internal stack trace tidak boleh dikembalikan.

## 5. Validation Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed.",
    "fields": [
      {
        "field": "items[0].quantity",
        "message": "Quantity must be greater than zero."
      }
    ]
  },
  "meta": {
    "request_id": "req_123",
    "correlation_id": "corr_123",
    "timestamp": "2026-05-27T10:00:00Z"
  }
}
```

## 6. Metadata

Setiap response harus menyertakan:

| Field | Deskripsi |
| --- | --- |
| `request_id` | Unique ID yang dibuat per HTTP request. |
| `correlation_id` | ID untuk melacak workflow lintas service. |
| `timestamp` | Response timestamp dalam format RFC3339/ISO-8601. |

Jika client mengirim `X-Correlation-Id`, server harus menggunakannya kembali. Jika tidak, server membuat value baru.

## 7. Pagination

Default pagination menggunakan page-based pagination.

Query parameter:

| Parameter | Default | Max | Deskripsi |
| --- | --- | --- | --- |
| `page` | `1` | - | Nomor page, dimulai dari 1. |
| `per_page` | `20` | `100` | Jumlah item per page. |

Pagination response:

```json
{
  "page": 1,
  "per_page": 20,
  "total_items": 120,
  "total_pages": 6,
  "has_next": true,
  "has_prev": false
}
```

Aturan:

- `page` kurang dari 1 harus diperlakukan sebagai validation error.
- `per_page` lebih besar dari max harus dicap atau direject secara konsisten. Rekomendasi: reject dengan `VALIDATION_ERROR`.
- Sort order harus deterministic.

## 8. Sorting

Gunakan format query ini:

```text
sort=created_at:desc
sort=price:asc
```

Aturan:

- Unknown sort field harus mengembalikan validation error.
- Unknown sort direction harus mengembalikan validation error.
- Default sort untuk order adalah `created_at:desc`.
- Default sort untuk product adalah `name:asc`.

## 9. Filtering dan Search

Product list may support:

```text
GET /products?category=semen&search=tiga&page=1&per_page=20&sort=price:asc
```

Order list may support:

```text
GET /orders?status=CONFIRMED&page=1&per_page=20&sort=created_at:desc
```

Aturan:

- Filter harus menggunakan explicit allowlist.
- Unknown filter harus mengembalikan validation error.
- Search sebaiknya case-insensitive jika praktis.

## 10. Konvensi Error Code

Gunakan uppercase snake case.

Error code umum:

| HTTP Status | Code |
| --- | --- |
| 400 | `VALIDATION_ERROR` |
| 400 | `INVALID_REQUEST` |
| 404 | `RESOURCE_NOT_FOUND` |
| 409 | `INSUFFICIENT_STOCK` |
| 409 | `INVALID_STATE_TRANSITION` |
| 409 | `DUPLICATE_REQUEST` |
| 500 | `INTERNAL_ERROR` |
| 503 | `SERVICE_UNAVAILABLE` |
