COMPOSE ?= podman compose
GO ?= go
GOFLAGS ?= -mod=readonly
GOOSE ?= goose
SQLC ?= sqlc

ORDER_DIR := services/order-service
INVENTORY_DIR := services/catalog-inventory-service
PAYMENT_DIR := services/payment-service
PROTO_FILES := proto/inventory/v1/inventory.proto proto/order/v1/order.proto proto/payment/v1/payment.proto
INVENTORY_DATABASE_URL ?= postgres://toko:toko@localhost:5432/inventory_db?sslmode=disable
PAYMENT_DATABASE_URL ?= postgres://toko:toko@localhost:5432/payment_db?sslmode=disable
ORDER_DATABASE_URL ?= postgres://toko:toko@localhost:5432/order_db?sslmode=disable

.PHONY: infra-up infra-down infra-logs infra-ps up down \
	order inventory payment \
	order-run inventory-run payment-run \
	proto-test shared-test order-test inventory-test payment-test test-all \
	order-build inventory-build payment-build build-all \
	order-migrate inventory-migrate payment-migrate migrate \
	order-migrate-compose inventory-migrate-compose payment-migrate-compose inventory-seed seed \
	proto proto-validate sqlc generate test

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
	cd $(ORDER_DIR) && DATABASE_URL="$(ORDER_DATABASE_URL)" $(GO) run $(GOFLAGS) ./cmd/api

inventory-run:
	cd $(INVENTORY_DIR) && DATABASE_URL="$(INVENTORY_DATABASE_URL)" $(GO) run $(GOFLAGS) ./cmd/api

payment-run:
	cd $(PAYMENT_DIR) && DATABASE_URL="$(PAYMENT_DATABASE_URL)" $(GO) run $(GOFLAGS) ./cmd/api

shared-test:
	cd shared && $(GO) test $(GOFLAGS) ./...

proto-test:
	cd proto && $(GO) test $(GOFLAGS) ./...

order-test:
	cd $(ORDER_DIR) && $(GO) test $(GOFLAGS) ./...

inventory-test:
	cd $(INVENTORY_DIR) && $(GO) test $(GOFLAGS) ./...

payment-test:
	cd $(PAYMENT_DIR) && $(GO) test $(GOFLAGS) ./...

test-all: proto-test shared-test order-test inventory-test payment-test

test: test-all

order-build:
	cd $(ORDER_DIR) && $(GO) build $(GOFLAGS) -o ../../bin/order-service ./cmd/api

inventory-build:
	cd $(INVENTORY_DIR) && $(GO) build $(GOFLAGS) -o ../../bin/catalog-inventory-service ./cmd/api

payment-build:
	cd $(PAYMENT_DIR) && $(GO) build $(GOFLAGS) -o ../../bin/payment-service ./cmd/api

build-all: order-build inventory-build payment-build

order-migrate:
	cd $(ORDER_DIR) && $(GOOSE) postgres "$$ORDER_DATABASE_URL" up

order-migrate-compose:
	awk '/-- \+goose Down/{exit} {print}' $(ORDER_DIR)/migrations/00001_create_order_tables.sql | $(COMPOSE) exec -T postgres psql -U toko -d order_db

inventory-migrate:
	cd $(INVENTORY_DIR) && $(GOOSE) postgres "$$INVENTORY_DATABASE_URL" up

inventory-migrate-compose:
	awk '/-- \+goose Down/{exit} {print}' $(INVENTORY_DIR)/migrations/00001_create_inventory_tables.sql | $(COMPOSE) exec -T postgres psql -U toko -d inventory_db

payment-migrate:
	cd $(PAYMENT_DIR) && $(GOOSE) postgres "$$PAYMENT_DATABASE_URL" up

payment-migrate-compose:
	awk '/-- \+goose Down/{exit} {print}' $(PAYMENT_DIR)/migrations/00001_create_payment_tables.sql | $(COMPOSE) exec -T postgres psql -U toko -d payment_db

migrate: order-migrate inventory-migrate payment-migrate

inventory-seed:
	$(COMPOSE) exec -T postgres psql -U toko -d inventory_db < $(INVENTORY_DIR)/seeds/001_demo_products.sql

seed: inventory-seed

proto:
	protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative $(PROTO_FILES)

proto-validate:
	protoc --proto_path=proto --descriptor_set_out=/tmp/toko-bangunan-proto.pb $(PROTO_FILES)

sqlc:
	cd $(ORDER_DIR) && $(SQLC) generate
	cd $(INVENTORY_DIR) && $(SQLC) generate

generate: proto sqlc
