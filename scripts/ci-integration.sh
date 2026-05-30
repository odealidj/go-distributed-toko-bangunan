#!/usr/bin/env bash
set -euo pipefail

for command in curl make; do
  if ! command -v "${command}" >/dev/null 2>&1; then
    echo "${command} wajib tersedia" >&2
    exit 1
  fi
done

read -r -a COMPOSE_CMD <<<"${COMPOSE:-docker compose}"
AUTO_CLEANUP="${CI_INTEGRATION_AUTO_CLEANUP:-1}"
ORDER_READY_URL="${ORDER_READY_URL:-http://localhost:8080/readyz}"
CATALOG_READY_URL="${CATALOG_READY_URL:-http://localhost:8081/readyz}"
PAYMENT_READY_URL="${PAYMENT_READY_URL:-http://localhost:8082/readyz}"

compose() {
  "${COMPOSE_CMD[@]}" "$@"
}

cleanup() {
  if [[ "${AUTO_CLEANUP}" == "1" ]]; then
    compose down -v --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

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

wait_http_ok() {
  local url="$1"

  retry 60 2 bash -lc "
    status=\$(curl -sS -o /tmp/toko-wait-body.\$\$ -w '%{http_code}' '${url}' || true)
    rm -f /tmp/toko-wait-body.\$\$
    [[ \"\${status}\" == \"200\" ]]
  "
}

wait_app_stack() {
  wait_http_ok "${ORDER_READY_URL}"
  wait_http_ok "${CATALOG_READY_URL}"
  wait_http_ok "${PAYMENT_READY_URL}"
}

wait_postgres_ready() {
  retry 45 2 compose exec -T postgres pg_isready -U toko -d order_db >/dev/null 2>&1
}

echo "[1/7] reset compose state"
compose down -v --remove-orphans >/dev/null 2>&1 || true

echo "[2/7] start infra"
make infra-up COMPOSE="${COMPOSE:-docker compose}"
wait_postgres_ready

echo "[3/7] migrate database dan buat topic"
make order-migrate-compose inventory-migrate-compose payment-migrate-compose COMPOSE="${COMPOSE:-docker compose}"
retry 10 3 make kafka-topics COMPOSE="${COMPOSE:-docker compose}"

echo "[4/7] start app services"
compose --profile app up -d --build
wait_app_stack

echo "[5/7] seed inventory dan jalankan integration test"
make inventory-seed COMPOSE="${COMPOSE:-docker compose}"
make test-integration COMPOSE="${COMPOSE:-docker compose}"
wait_app_stack

echo "[6/7] jalankan e2e test"
make test-e2e COMPOSE="${COMPOSE:-docker compose}"

echo "[7/7] selesai"
