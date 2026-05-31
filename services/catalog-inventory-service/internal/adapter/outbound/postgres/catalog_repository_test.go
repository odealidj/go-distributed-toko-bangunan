package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
)

func TestCatalogRepositoryIntegration_DuplicateOrderCancelledDoesNotReleaseTwice(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL tidak diset")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			inbox_events,
			outbox_events,
			stock_reservation_items,
			stock_reservations,
			inventories,
			products,
			categories
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, is_active)
		VALUES ('cat_test_repo', 'Kategori Test', true)
	`); err != nil {
		t.Fatalf("insert category: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO products (id, category_id, sku, name, unit, price, weight_kg, requires_truck, is_active)
		VALUES ('prod_test_repo', 'cat_test_repo', 'SKU-TEST-1', 'Produk Test', 'sak', 100000, 50, false, true)
	`); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO inventories (product_id, on_hand_qty, reserved_qty)
		VALUES ('prod_test_repo', 20, 0)
	`); err != nil {
		t.Fatalf("insert inventory: %v", err)
	}

	repository := NewCatalogRepository(pool)
	reservation, err := repository.ReserveStock(ctx, model.ReserveStockCommand{
		OrderID:        "ord_test_inventory_1",
		IdempotencyKey: "reserve-stock:ord_test_inventory_1",
		Items: []model.OrderItemInput{
			{ProductID: "prod_test_repo", Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("ReserveStock: %v", err)
	}
	if reservation.Status != model.ReservationStatusReserved {
		t.Fatalf("reservation status = %s, want %s", reservation.Status, model.ReservationStatusReserved)
	}

	duplicate, err := repository.ProcessOrderEvent(ctx, messaging.EventEnvelope{
		EventID:       "evt_test_inventory_cancelled",
		EventType:     "OrderCancelled",
		AggregateID:   "ord_test_inventory_1",
		AggregateType: "order",
		CorrelationID: "corr_test_inventory_1",
	})
	if err != nil {
		t.Fatalf("ProcessOrderEvent first call: %v", err)
	}
	if duplicate {
		t.Fatal("first event should not be duplicate")
	}

	duplicate, err = repository.ProcessOrderEvent(ctx, messaging.EventEnvelope{
		EventID:       "evt_test_inventory_cancelled",
		EventType:     "OrderCancelled",
		AggregateID:   "ord_test_inventory_1",
		AggregateType: "order",
		CorrelationID: "corr_test_inventory_1",
	})
	if err != nil {
		t.Fatalf("ProcessOrderEvent duplicate call: %v", err)
	}
	if !duplicate {
		t.Fatal("duplicate event should be detected")
	}

	var reservationStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM stock_reservations WHERE order_id = $1`, "ord_test_inventory_1").Scan(&reservationStatus); err != nil {
		t.Fatalf("query reservation status: %v", err)
	}
	if reservationStatus != model.ReservationStatusReleased {
		t.Fatalf("reservation status = %s, want %s", reservationStatus, model.ReservationStatusReleased)
	}

	var onHand string
	var reserved string
	if err := pool.QueryRow(ctx, `SELECT on_hand_qty::text, reserved_qty::text FROM inventories WHERE product_id = $1`, "prod_test_repo").Scan(&onHand, &reserved); err != nil {
		t.Fatalf("query inventory quantities: %v", err)
	}
	if onHand != "20" && onHand != "20.0000" {
		t.Fatalf("on_hand_qty = %s, want 20", onHand)
	}
	if reserved != "0" && reserved != "0.0000" {
		t.Fatalf("reserved_qty = %s, want 0", reserved)
	}

	var inboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_events WHERE event_id = $1`, "evt_test_inventory_cancelled").Scan(&inboxCount); err != nil {
		t.Fatalf("query inbox count: %v", err)
	}
	if inboxCount != 1 {
		t.Fatalf("inbox count = %d, want 1", inboxCount)
	}
}
