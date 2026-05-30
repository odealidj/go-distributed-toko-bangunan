#!/usr/bin/env bash
set -euo pipefail

for command in curl jq awk; do
  if ! command -v "${command}" >/dev/null 2>&1; then
    echo "${command} wajib tersedia" >&2
    exit 1
  fi
done

read -r -a COMPOSE_CMD <<<"${COMPOSE:-podman compose}"

ORDER_BASE_URL="${ORDER_BASE_URL:-http://localhost:8080}"
CATALOG_BASE_URL="${CATALOG_BASE_URL:-http://localhost:8081}"
PAYMENT_BASE_URL="${PAYMENT_BASE_URL:-http://localhost:8082}"
PRODUCT_ID="${PRODUCT_ID:-prod_semen_50kg}"
SUCCESS_QTY="${SUCCESS_QTY:-2}"
FAILURE_QTY="${FAILURE_QTY:-2}"
INSUFFICIENT_QTY="${INSUFFICIENT_QTY:-10000}"
DUPLICATE_EVENT_RETRY_ATTEMPTS="${DUPLICATE_EVENT_RETRY_ATTEMPTS:-30}"
MANUAL_SETTLE_RETRY_ATTEMPTS="${MANUAL_SETTLE_RETRY_ATTEMPTS:-30}"
REDIS_FALLBACK_RETRY_ATTEMPTS="${REDIS_FALLBACK_RETRY_ATTEMPTS:-15}"

redis_stopped=0

cleanup() {
  if [[ "${redis_stopped}" == "1" ]]; then
    compose start redis >/dev/null
  fi
}
trap cleanup EXIT

compose() {
  "${COMPOSE_CMD[@]}" "$@"
}

db_query() {
  local database="$1"
  local sql="$2"
  compose exec -T postgres psql -U toko -d "${database}" -Atqc "${sql}"
}

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  if [[ "${expected}" != "${actual}" ]]; then
    echo "assertion failed: ${message}. expected=${expected} actual=${actual}" >&2
    exit 1
  fi
}

assert_true() {
  local value="$1"
  local message="$2"
  if [[ "${value}" != "true" ]]; then
    echo "assertion failed: ${message}" >&2
    exit 1
  fi
}

assert_http_ok() {
  local url="$1"
  local message="$2"
  local status
  status="$(curl -sS -o /tmp/toko-http-body.$$ -w '%{http_code}' "${url}")"
  if [[ "${status}" != "200" ]]; then
    echo "assertion failed: ${message}. http_status=${status}" >&2
    cat /tmp/toko-http-body.$$ >&2
    rm -f /tmp/toko-http-body.$$
    exit 1
  fi
  rm -f /tmp/toko-http-body.$$
}

retry() {
  local attempts="$1"
  local sleep_seconds="$2"
  shift 2

  local attempt
  for ((attempt=1; attempt<=attempts; attempt++)); do
    if "$@"; then
      return 0
    fi
    if [[ "${attempt}" -lt "${attempts}" ]]; then
      sleep "${sleep_seconds}"
    fi
  done
  return 1
}

retry_silent() {
  local attempts="$1"
  local sleep_seconds="$2"
  shift 2

  retry "${attempts}" "${sleep_seconds}" "$@" >/dev/null 2>&1
}

decimal_sub() {
  local left="$1"
  local right="$2"
  awk "BEGIN { printf \"%.4f\", (${left}) - (${right}) }"
}

decimal_trim() {
  local value="$1"
  awk "BEGIN { printf \"%.4f\", (${value}) }"
}

dump_debug_snapshot() {
  local label="$1"
  local order_id="${2:-}"
  local payment_id="${3:-}"

  echo "[debug] ${label}" >&2
  echo "[debug] compose ps" >&2
  compose ps >&2 || true

  if [[ -n "${order_id}" ]]; then
    echo "[debug] order ${order_id}" >&2
    db_query order_db "SELECT id, status, payment_id, correlation_id, updated_at FROM orders WHERE id = '${order_id}'" >&2 || true
    db_query order_db "SELECT event_type, status, aggregate_id, correlation_id, created_at FROM outbox_events WHERE aggregate_id = '${order_id}' ORDER BY created_at" >&2 || true
  fi

  if [[ -n "${payment_id}" ]]; then
    echo "[debug] payment ${payment_id}" >&2
    db_query payment_db "SELECT id, order_id, status, payment_mode, correlation_id, updated_at FROM payments WHERE id = '${payment_id}'" >&2 || true
    db_query payment_db "SELECT event_type, status, aggregate_id, correlation_id, created_at FROM outbox_events WHERE aggregate_id = '${payment_id}' ORDER BY created_at" >&2 || true
  fi

  if [[ -n "${order_id}" ]]; then
    echo "[debug] reservation ${order_id}" >&2
    db_query inventory_db "SELECT order_id, status, reserved_qty, updated_at FROM stock_reservations WHERE order_id = '${order_id}'" >&2 || true
  fi

  echo "[debug] inventory ${PRODUCT_ID}" >&2
  db_query inventory_db "SELECT product_id, available_qty, reserved_qty, on_hand_qty, updated_at FROM inventories WHERE product_id = '${PRODUCT_ID}'" >&2 || true
}

post_order() {
  local payment_mode="$1"
  local quantity="$2"
  local correlation_id="corr_e2e_$(date +%s%N)"
  local request_id="req_e2e_$(date +%s%N)"
  local payload

  payload="$(jq -cn \
    --arg payment_mode "${payment_mode}" \
    --arg product_id "${PRODUCT_ID}" \
    --argjson quantity "${quantity}" \
    '{
      customer: {
        name: "Tester CLI",
        phone: "081234567890",
        address: "Jl. E2E No. 7"
      },
      payment_mode: $payment_mode,
      items: [{product_id: $product_id, quantity: $quantity}]
    }'
  )"

  curl -fsS \
    -H "Content-Type: application/json" \
    -H "X-Request-Id: ${request_id}" \
    -H "X-Correlation-Id: ${correlation_id}" \
    -d "${payload}" \
    "${ORDER_BASE_URL}/orders"
}

post_order_response() {
  post_order "$1" "$2"
}

post_order_cancel() {
  local order_id="$1"
  local correlation_id="corr_cancel_$(date +%s%N)"
  local request_id="req_cancel_$(date +%s%N)"

  curl -fsS \
    -X POST \
    -H "X-Request-Id: ${request_id}" \
    -H "X-Correlation-Id: ${correlation_id}" \
    "${ORDER_BASE_URL}/orders/${order_id}/cancel"
}

post_payment_succeed() {
  local payment_id="$1"
  local correlation_id="corr_pay_succeed_$(date +%s%N)"
  local request_id="req_pay_succeed_$(date +%s%N)"

  curl -fsS \
    -X POST \
    -H "X-Request-Id: ${request_id}" \
    -H "X-Correlation-Id: ${correlation_id}" \
    "${PAYMENT_BASE_URL}/demo/payments/${payment_id}/succeed"
}

post_payment_fail() {
  local payment_id="$1"
  local correlation_id="corr_pay_fail_$(date +%s%N)"
  local request_id="req_pay_fail_$(date +%s%N)"

  curl -fsS \
    -X POST \
    -H "X-Request-Id: ${request_id}" \
    -H "X-Correlation-Id: ${correlation_id}" \
    "${PAYMENT_BASE_URL}/demo/payments/${payment_id}/fail"
}

publish_duplicate_order_event() {
  local event_id="$1"
  local event_type="$2"
  local aggregate_id="$3"
  local occurred_at="$4"
  local correlation_id="$5"
  local causation_id="$6"
  local traceparent="$7"
  local payload_json="$8"

  local envelope
  envelope="$(jq -cn \
    --arg event_id "${event_id}" \
    --arg event_type "${event_type}" \
    --arg aggregate_id "${aggregate_id}" \
    --arg occurred_at "${occurred_at}" \
    --arg correlation_id "${correlation_id}" \
    --arg causation_id "${causation_id}" \
    --argjson payload "${payload_json}" \
    '{
      event_id: $event_id,
      event_type: $event_type,
      aggregate_id: $aggregate_id,
      aggregate_type: "order",
      occurred_at: $occurred_at,
      correlation_id: $correlation_id,
      causation_id: $causation_id,
      payload: $payload
    }'
  )"

  local headers="x-correlation-id:${correlation_id},x-causation-id:${causation_id},x-event-id:${event_id},x-event-type:${event_type}"
  if [[ -n "${traceparent}" ]]; then
    headers+=",traceparent:${traceparent}"
  fi

  printf '%s\t%s\t%s\n' "${headers}" "${aggregate_id}" "${envelope}" | \
    compose exec -T kafka /opt/kafka/bin/kafka-console-producer.sh \
      --bootstrap-server kafka:9092 \
      --topic order.events \
      --sync \
      --property parse.headers=true \
      --property parse.key=true
}

seed_inventory_demo() {
  compose exec -T postgres psql -U toko -d inventory_db < services/catalog-inventory-service/seeds/001_demo_products.sql >/dev/null
}

duplicate_event_is_settled() {
  local duplicate_event_id="$1"
  local duplicate_before_on_hand="$2"

  local duplicate_after_on_hand
  local inventory_inbox_count
  local payment_inbox_count

  duplicate_after_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
  inventory_inbox_count="$(db_query inventory_db "SELECT count(*) FROM inbox_events WHERE event_id = '${duplicate_event_id}'")"
  payment_inbox_count="$(db_query payment_db "SELECT count(*) FROM inbox_events WHERE event_id = '${duplicate_event_id}'")"

  [[ "$(decimal_trim "${duplicate_before_on_hand}")" == "$(decimal_trim "${duplicate_after_on_hand}")" ]] || return 1
  [[ "${inventory_inbox_count}" == "1" ]] || return 1
  [[ "${payment_inbox_count}" == "1" ]] || return 1
}

manual_payment_success_is_settled() {
  local order_id="$1"
  local payment_id="$2"
  local expected_on_hand="$3"

  local order_status
  local payment_status
  local reservation_status
  local on_hand

  order_status="$(db_query order_db "SELECT status FROM orders WHERE id = '${order_id}'")"
  payment_status="$(db_query payment_db "SELECT status FROM payments WHERE id = '${payment_id}'")"
  reservation_status="$(db_query inventory_db "SELECT status FROM stock_reservations WHERE order_id = '${order_id}'")"
  on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"

  [[ "${order_status}" == "CONFIRMED" ]] || return 1
  [[ "${payment_status}" == "SUCCEEDED" ]] || return 1
  [[ "${reservation_status}" == "COMMITTED" ]] || return 1
  [[ "$(decimal_trim "${expected_on_hand}")" == "$(decimal_trim "${on_hand}")" ]] || return 1
}

manual_payment_failed_is_settled() {
  local order_id="$1"
  local payment_id="$2"
  local expected_on_hand="$3"

  local order_status
  local payment_status
  local reservation_status
  local on_hand

  order_status="$(db_query order_db "SELECT status FROM orders WHERE id = '${order_id}'")"
  payment_status="$(db_query payment_db "SELECT status FROM payments WHERE id = '${payment_id}'")"
  reservation_status="$(db_query inventory_db "SELECT status FROM stock_reservations WHERE order_id = '${order_id}'")"
  on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"

  [[ "${order_status}" == "CANCELLED" ]] || return 1
  [[ "${payment_status}" == "FAILED" ]] || return 1
  [[ "${reservation_status}" == "RELEASED" ]] || return 1
  [[ "$(decimal_trim "${expected_on_hand}")" == "$(decimal_trim "${on_hand}")" ]] || return 1
}

manual_order_cancel_is_settled() {
  local order_id="$1"
  local payment_id="$2"
  local expected_on_hand="$3"

  local order_status
  local payment_status
  local reservation_status
  local on_hand

  order_status="$(db_query order_db "SELECT status FROM orders WHERE id = '${order_id}'")"
  payment_status="$(db_query payment_db "SELECT status FROM payments WHERE id = '${payment_id}'")"
  reservation_status="$(db_query inventory_db "SELECT status FROM stock_reservations WHERE order_id = '${order_id}'")"
  on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"

  [[ "${order_status}" == "CANCELLED" ]] || return 1
  [[ "${payment_status}" == "CANCELLED" ]] || return 1
  [[ "${reservation_status}" == "RELEASED" ]] || return 1
  [[ "$(decimal_trim "${expected_on_hand}")" == "$(decimal_trim "${on_hand}")" ]] || return 1
}

echo "[setup] seed inventory demo dan cek readiness"
seed_inventory_demo
assert_http_ok "${ORDER_BASE_URL}/readyz" "order-service harus ready"
assert_http_ok "${CATALOG_BASE_URL}/readyz" "catalog-inventory-service harus ready"
assert_http_ok "${PAYMENT_BASE_URL}/readyz" "payment-service harus ready"

echo "[1/5] checkout success"
success_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
success_response="$(post_order SUCCESS "${SUCCESS_QTY}")"
success_order_id="$(jq -r '.data.id' <<<"${success_response}")"
success_status="$(jq -r '.data.status' <<<"${success_response}")"
success_payment_id="$(jq -r '.data.payment_id' <<<"${success_response}")"
assert_equals "CONFIRMED" "${success_status}" "status order success"

success_payment_status="$(db_query payment_db "SELECT status FROM payments WHERE order_id = '${success_order_id}'")"
success_reservation_status="$(db_query inventory_db "SELECT status FROM stock_reservations WHERE order_id = '${success_order_id}'")"
success_after_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
success_expected_on_hand="$(decimal_sub "${success_before_on_hand}" "${SUCCESS_QTY}")"
assert_equals "SUCCEEDED" "${success_payment_status}" "status payment success"
assert_equals "COMMITTED" "${success_reservation_status}" "status reservasi success"
assert_equals "$(decimal_trim "${success_expected_on_hand}")" "$(decimal_trim "${success_after_on_hand}")" "on_hand setelah success"

echo "[2/5] insufficient stock"
insufficient_response="$(post_order SUCCESS "${INSUFFICIENT_QTY}")"
insufficient_order_id="$(jq -r '.data.id' <<<"${insufficient_response}")"
insufficient_status="$(jq -r '.data.status' <<<"${insufficient_response}")"
assert_equals "REJECTED" "${insufficient_status}" "status order insufficient stock"
insufficient_payment_count="$(db_query payment_db "SELECT count(*) FROM payments WHERE order_id = '${insufficient_order_id}'")"
insufficient_reservation_count="$(db_query inventory_db "SELECT count(*) FROM stock_reservations WHERE order_id = '${insufficient_order_id}'")"
assert_equals "0" "${insufficient_payment_count}" "payment tidak boleh dibuat saat stock kurang"
assert_equals "0" "${insufficient_reservation_count}" "reservasi tidak boleh dibuat saat stock kurang"

echo "[3/5] payment failed compensation"
failure_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
failure_response="$(post_order FAILURE "${FAILURE_QTY}")"
failure_order_id="$(jq -r '.data.id' <<<"${failure_response}")"
failure_status="$(jq -r '.data.status' <<<"${failure_response}")"
failure_payment_id="$(jq -r '.data.payment_id' <<<"${failure_response}")"
assert_equals "CANCELLED" "${failure_status}" "status order payment failure"

failure_payment_status="$(db_query payment_db "SELECT status FROM payments WHERE id = '${failure_payment_id}'")"
failure_reservation_status="$(db_query inventory_db "SELECT status FROM stock_reservations WHERE order_id = '${failure_order_id}'")"
failure_after_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
assert_equals "FAILED" "${failure_payment_status}" "status payment failure"
assert_equals "RELEASED" "${failure_reservation_status}" "status reservasi compensation"
assert_equals "$(decimal_trim "${failure_before_on_hand}")" "$(decimal_trim "${failure_after_on_hand}")" "on_hand harus kembali setelah compensation"

echo "[4/5] duplicate event idempotency"
duplicate_event_row="$(db_query order_db "SELECT id, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.MS\"Z\"'), correlation_id, coalesce(causation_id, ''), coalesce(traceparent, ''), payload::text FROM outbox_events WHERE aggregate_id = '${success_order_id}' AND event_type = 'OrderConfirmed' ORDER BY created_at DESC LIMIT 1")"
IFS='|' read -r duplicate_event_id duplicate_occurred_at duplicate_correlation_id duplicate_causation_id duplicate_traceparent duplicate_payload <<<"$(sed 's/\t/|/g' <<<"${duplicate_event_row}")"
duplicate_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"

publish_duplicate_order_event \
  "${duplicate_event_id}" \
  "OrderConfirmed" \
  "${success_order_id}" \
  "${duplicate_occurred_at}" \
  "${duplicate_correlation_id}" \
  "${duplicate_causation_id}" \
  "${duplicate_traceparent}" \
  "${duplicate_payload}"

retry "${DUPLICATE_EVENT_RETRY_ATTEMPTS}" 1 duplicate_event_is_settled "${duplicate_event_id}" "${duplicate_before_on_hand}" || {
  echo "assertion failed: duplicate event belum settle dalam waktu yang diharapkan" >&2
  dump_debug_snapshot "duplicate event belum settle" "${success_order_id}" "${success_payment_id}"
  exit 1
}

duplicate_after_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
inventory_inbox_count="$(db_query inventory_db "SELECT count(*) FROM inbox_events WHERE event_id = '${duplicate_event_id}'")"
payment_inbox_count="$(db_query payment_db "SELECT count(*) FROM inbox_events WHERE event_id = '${duplicate_event_id}'")"
assert_equals "$(decimal_trim "${duplicate_before_on_hand}")" "$(decimal_trim "${duplicate_after_on_hand}")" "duplicate event tidak boleh mengurangi stock lagi"
assert_equals "1" "${inventory_inbox_count}" "inventory inbox harus tetap satu event"
assert_equals "1" "${payment_inbox_count}" "payment inbox harus tetap satu event"

echo "[5/8] manual payment success"
manual_success_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
manual_success_response="$(post_order_response MANUAL 1)"
manual_success_order_id="$(jq -r '.data.id' <<<"${manual_success_response}")"
manual_success_status="$(jq -r '.data.status' <<<"${manual_success_response}")"
manual_success_payment_id="$(jq -r '.data.payment_id' <<<"${manual_success_response}")"
assert_equals "WAITING_PAYMENT" "${manual_success_status}" "status order manual success harus waiting payment"

post_payment_succeed "${manual_success_payment_id}" >/dev/null

manual_success_expected_on_hand="$(decimal_sub "${manual_success_before_on_hand}" "1")"
retry "${MANUAL_SETTLE_RETRY_ATTEMPTS}" 1 manual_payment_success_is_settled "${manual_success_order_id}" "${manual_success_payment_id}" "${manual_success_expected_on_hand}" || {
  echo "assertion failed: manual payment success belum settle dalam waktu yang diharapkan" >&2
  dump_debug_snapshot "manual payment success belum settle" "${manual_success_order_id}" "${manual_success_payment_id}"
  exit 1
}

echo "[6/8] manual payment failed"
manual_fail_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
manual_fail_response="$(post_order_response MANUAL 1)"
manual_fail_order_id="$(jq -r '.data.id' <<<"${manual_fail_response}")"
manual_fail_status="$(jq -r '.data.status' <<<"${manual_fail_response}")"
manual_fail_payment_id="$(jq -r '.data.payment_id' <<<"${manual_fail_response}")"
assert_equals "WAITING_PAYMENT" "${manual_fail_status}" "status order manual failure harus waiting payment"

post_payment_fail "${manual_fail_payment_id}" >/dev/null

retry "${MANUAL_SETTLE_RETRY_ATTEMPTS}" 1 manual_payment_failed_is_settled "${manual_fail_order_id}" "${manual_fail_payment_id}" "${manual_fail_before_on_hand}" || {
  echo "assertion failed: manual payment failed belum settle dalam waktu yang diharapkan" >&2
  dump_debug_snapshot "manual payment failed belum settle" "${manual_fail_order_id}" "${manual_fail_payment_id}"
  exit 1
}

echo "[7/8] manual order cancel"
manual_cancel_before_on_hand="$(db_query inventory_db "SELECT on_hand_qty::text FROM inventories WHERE product_id = '${PRODUCT_ID}'")"
manual_cancel_response="$(post_order_response MANUAL 1)"
manual_cancel_order_id="$(jq -r '.data.id' <<<"${manual_cancel_response}")"
manual_cancel_status="$(jq -r '.data.status' <<<"${manual_cancel_response}")"
manual_cancel_payment_id="$(jq -r '.data.payment_id' <<<"${manual_cancel_response}")"
assert_equals "WAITING_PAYMENT" "${manual_cancel_status}" "status order manual cancel harus waiting payment"

post_order_cancel "${manual_cancel_order_id}" >/dev/null

retry "${MANUAL_SETTLE_RETRY_ATTEMPTS}" 1 manual_order_cancel_is_settled "${manual_cancel_order_id}" "${manual_cancel_payment_id}" "${manual_cancel_before_on_hand}" || {
  echo "assertion failed: manual order cancel belum settle dalam waktu yang diharapkan" >&2
  dump_debug_snapshot "manual order cancel belum settle" "${manual_cancel_order_id}" "${manual_cancel_payment_id}"
  exit 1
}

echo "[8/8] redis unavailable fallback"
compose stop redis >/dev/null
redis_stopped=1

retry "${REDIS_FALLBACK_RETRY_ATTEMPTS}" 1 assert_http_ok "${CATALOG_BASE_URL}/products/${PRODUCT_ID}" "GET product saat redis down harus fallback ke PostgreSQL"
redis_response="$(post_order SUCCESS 1)"
redis_order_status="$(jq -r '.data.status' <<<"${redis_response}")"
assert_equals "CONFIRMED" "${redis_order_status}" "checkout tetap harus berhasil saat redis down"

compose start redis >/dev/null
redis_stopped=0

echo "semua scenario E2E berhasil"
