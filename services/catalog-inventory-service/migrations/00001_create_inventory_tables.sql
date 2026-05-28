-- +goose Up
CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE products (
    id TEXT PRIMARY KEY,
    category_id TEXT NOT NULL REFERENCES categories(id),
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    brand TEXT,
    unit TEXT NOT NULL,
    price BIGINT NOT NULL,
    weight_kg NUMERIC(18, 4),
    requires_truck BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE inventories (
    product_id TEXT PRIMARY KEY REFERENCES products(id),
    on_hand_qty NUMERIC(18, 4) NOT NULL,
    reserved_qty NUMERIC(18, 4) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE stock_reservations (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE stock_reservation_items (
    id TEXT PRIMARY KEY,
    reservation_id TEXT NOT NULL REFERENCES stock_reservations(id),
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity NUMERIC(18, 4) NOT NULL
);

CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    causation_id TEXT,
    traceparent TEXT,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE TABLE inbox_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    traceparent TEXT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_products_category_active ON products(category_id, is_active);
CREATE INDEX idx_outbox_events_status_created_at ON outbox_events(status, created_at);

-- +goose Down
DROP TABLE IF EXISTS inbox_events;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS stock_reservation_items;
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS inventories;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;

