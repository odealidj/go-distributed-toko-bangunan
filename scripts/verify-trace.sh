#!/usr/bin/env bash
set -euo pipefail

if ! command -v curl >/dev/null 2>&1; then
  echo "curl wajib tersedia" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq wajib tersedia untuk verifikasi trace" >&2
  exit 1
fi

ORDER_URL="${ORDER_URL:-http://localhost:8080/orders}"
JAEGER_API_URL="${JAEGER_API_URL:-http://localhost:16686/api/traces}"
PAYMENT_MODE="${PAYMENT_MODE:-SUCCESS}"
PRODUCT_ID="${PRODUCT_ID:-prod_semen_50kg}"
QUANTITY="${QUANTITY:-2}"
TRACE_WAIT_SECONDS="${TRACE_WAIT_SECONDS:-5}"
CORRELATION_ID="${CORRELATION_ID:-corr_demo_$(date +%s)}"
REQUEST_ID="${REQUEST_ID:-req_demo_$(date +%s)}"

payload="$(jq -cn \
  --arg payment_mode "$PAYMENT_MODE" \
  --arg product_id "$PRODUCT_ID" \
  --argjson quantity "$QUANTITY" \
  '{
    customer: {
      name: "Demo Observability",
      phone: "081234567890",
      address: "Jl. Tukang Bangunan No. 1"
    },
    payment_mode: $payment_mode,
    items: [{product_id: $product_id, quantity: $quantity}]
  }'
)"

echo "checkout order dengan correlation_id=${CORRELATION_ID}"
response="$(curl -fsS \
  -H "Content-Type: application/json" \
  -H "X-Request-Id: ${REQUEST_ID}" \
  -H "X-Correlation-Id: ${CORRELATION_ID}" \
  -d "${payload}" \
  "${ORDER_URL}")"

order_id="$(jq -r '.data.id' <<<"${response}")"
response_correlation_id="$(jq -r '.meta.correlation_id' <<<"${response}")"

if [[ -z "${order_id}" || "${order_id}" == "null" ]]; then
  echo "gagal mengambil order_id dari response checkout" >&2
  echo "${response}" >&2
  exit 1
fi

echo "order_id=${order_id}"
echo "menunggu trace masuk ke Jaeger ${TRACE_WAIT_SECONDS}s"
sleep "${TRACE_WAIT_SECONDS}"

tags="$(jq -cn --arg correlation_id "${response_correlation_id}" '{correlation_id: $correlation_id}')"
traces="$(curl -fsS --get \
  --data-urlencode "service=order-service" \
  --data-urlencode "lookback=1h" \
  --data-urlencode "limit=20" \
  --data-urlencode "tags=${tags}" \
  "${JAEGER_API_URL}")"

trace_count="$(jq '.data | length' <<<"${traces}")"
if [[ "${trace_count}" == "0" ]]; then
  echo "trace dengan correlation_id=${response_correlation_id} belum ditemukan di Jaeger" >&2
  exit 1
fi

trace_id="$(jq -r '.data[0].traceID' <<<"${traces}")"
services="$(jq -r '.data[0].processes | to_entries[] | .value.serviceName' <<<"${traces}" | sort -u)"

for service in order-service catalog-inventory-service payment-service; do
  if ! grep -qx "${service}" <<<"${services}"; then
    echo "trace ditemukan tetapi service ${service} belum ada di trace ${trace_id}" >&2
    echo "${services}" >&2
    exit 1
  fi
done

echo "trace_id=${trace_id}"
echo "services:"
echo "${services}"
echo "Jaeger UI: http://localhost:16686/trace/${trace_id}"
