# Desain Auth

## 1. Keputusan

Auth pada MVP/demo diimplementasikan sebagai middleware/edge concern, bukan sebagai microservice terpisah.

Tidak ada service berikut pada fase awal:

```text
auth-service
```

Service tetap:

```text
order-service
catalog-inventory-service
payment-service
```

## 2. Alasan

Fokus utama project adalah:

- Saga orchestration;
- Kafka event;
- gRPC internal sync;
- outbox/inbox;
- distributed transaction compensation;
- OpenTelemetry.

Membuat `auth-service` akan menambah scope:

- user registration;
- login;
- refresh token;
- password reset;
- token revocation;
- user database;
- inter-service auth.

Scope tersebut belum dibutuhkan untuk demo distributed transaction.

## 3. Model Auth MVP

Gunakan JWT.

Role:

```text
CUSTOMER
ADMIN
```

Claim minimal:

```json
{
  "sub": "usr_123",
  "role": "CUSTOMER",
  "iat": 1710000000,
  "exp": 1710003600
}
```

Header:

```text
Authorization: Bearer <jwt>
```

## 4. Penempatan Middleware

Middleware dapat ditempatkan di:

```text
services/order-service/internal/adapter/inbound/rest/middleware
services/catalog-inventory-service/internal/adapter/inbound/rest/middleware
```

Jika nanti API gateway dibuat, middleware dapat dipindahkan ke edge layer.

## 5. Endpoint Auth Demo

Untuk demo, boleh dibuat endpoint internal/demo:

```text
POST /demo/tokens/customer
POST /demo/tokens/admin
```

Endpoint ini menghasilkan JWT untuk testing/demo.

Endpoint demo token harus:

- hanya aktif saat `APP_ENV=local` atau `DEMO_MODE=true`;
- tidak digunakan untuk production.

## 6. Authorization Rule

| Endpoint | Auth | Role |
| --- | --- | --- |
| `GET /products` | Tidak wajib | Public |
| `GET /products/{product_id}` | Tidak wajib | Public |
| `POST /orders` | Wajib | `CUSTOMER` atau `ADMIN` |
| `GET /orders` | Wajib | `ADMIN` |
| `GET /orders/{order_id}` | Wajib | Owner atau `ADMIN` |
| `POST /orders/{order_id}/cancel` | Wajib | Owner atau `ADMIN` |
| `/demo/*` | Wajib/Local only | `ADMIN` atau local mode |

## 7. Error Response

JWT missing:

```text
HTTP 401
code: UNAUTHORIZED
```

JWT invalid/expired:

```text
HTTP 401
code: INVALID_TOKEN
```

Role tidak cukup:

```text
HTTP 403
code: FORBIDDEN
```

Response harus mengikuti:

```text
doc/api/response-standard.md
```

## 8. Context Propagation

Auth middleware harus menyimpan user context:

```text
user_id
role
```

Ke request context agar application use case dapat menggunakannya tanpa parsing JWT ulang.

## 9. Future Extraction

Jika auth berkembang, dapat dibuat service terpisah:

```text
auth-service
```

Trigger extraction:

- butuh user registration production;
- refresh token rotation;
- token revocation;
- multi-tenant user management;
- OAuth/OIDC integration;
- centralized authorization policy.

Sampai trigger tersebut muncul, auth tetap sebagai middleware/edge concern.

