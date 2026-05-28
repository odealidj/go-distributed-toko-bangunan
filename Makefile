COMPOSE ?= podman compose
GO ?= go
GOOSE ?= goose
SQLC ?= sqlc

ORDER_DIR := services/order-service
INVENTORY_DIR := services/catalog-inventory-service
PAYMENT_DIR := services/payment-service

.PHONY: infra-up infra-down infra-logs infra-ps up down \
	order inventory payment \
	order-run inventory-run payment-run \
	shared-test order-test inventory-test payment-test test-all \
	order-build inventory-build payment-build build-all \
	order-migrate inventory-migrate payment-migrate migrate \
	proto sqlc generate test

infra-up:
	$(COMPOSE) up -d postgres redis kafka kafka-ui otel-collector jaeger

infra-down:
	$(COMPOSE) down

infra-logs:
	$(COMPOSE) logs -f postgres redis kafka kafka-ui otel-collector jaeger

infra-ps:
	$(COMPOSE) ps

up:
	$(COMPOSE) --profile app up -d

down:
	$(COMPOSE) --profile app down

order: order-run

inventory: inventory-run

payment: payment-run

order-run:
	cd $(ORDER_DIR) && $(GO) run ./cmd/api

inventory-run:
	cd $(INVENTORY_DIR) && $(GO) run ./cmd/api

payment-run:
	cd $(PAYMENT_DIR) && $(GO) run ./cmd/api

shared-test:
	cd shared && $(GO) test ./...

order-test:
	cd $(ORDER_DIR) && $(GO) test ./...

inventory-test:
	cd $(INVENTORY_DIR) && $(GO) test ./...

payment-test:
	cd $(PAYMENT_DIR) && $(GO) test ./...

test-all: shared-test order-test inventory-test payment-test

test: test-all

order-build:
	cd $(ORDER_DIR) && $(GO) build -o ../../bin/order-service ./cmd/api

inventory-build:
	cd $(INVENTORY_DIR) && $(GO) build -o ../../bin/catalog-inventory-service ./cmd/api

payment-build:
	cd $(PAYMENT_DIR) && $(GO) build -o ../../bin/payment-service ./cmd/api

build-all: order-build inventory-build payment-build

order-migrate:
	cd $(ORDER_DIR) && $(GOOSE) postgres "$$ORDER_DATABASE_URL" up

inventory-migrate:
	cd $(INVENTORY_DIR) && $(GOOSE) postgres "$$INVENTORY_DATABASE_URL" up

payment-migrate:
	cd $(PAYMENT_DIR) && $(GOOSE) postgres "$$PAYMENT_DATABASE_URL" up

migrate: order-migrate inventory-migrate payment-migrate

proto:
	@echo "proto generation belum diaktifkan"

sqlc:
	cd $(ORDER_DIR) && $(SQLC) generate
	cd $(INVENTORY_DIR) && $(SQLC) generate

generate: proto sqlc
