package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/adapter/outbound/postgres/sqlc"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/application/port"
	"github.com/odealidj/go-distributed-toko-bangunan/services/catalog-inventory-service/internal/domain/model"
	"github.com/odealidj/go-distributed-toko-bangunan/shared/messaging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type CatalogRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var catalogRepositoryTracer = otel.Tracer("catalog-inventory-service/postgres")

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *CatalogRepository) Ping(ctx context.Context) error {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.Ping")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	if err := r.pool.Ping(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (r *CatalogRepository) ListProducts(ctx context.Context, filter model.ProductFilter) (port.ProductList, error) {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.ListProducts")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.Int("page", filter.Page),
		attribute.Int("per_page", filter.PerPage),
	)
	limit := filter.PerPage
	offset := (filter.Page - 1) * filter.PerPage

	rows, err := r.queries.ListProducts(ctx, sqlc.ListProductsParams{
		CategoryID: filter.CategoryID,
		Search:     filter.Search,
		OffsetRows: int32(offset),
		LimitRows:  int32(limit),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return port.ProductList{}, err
	}

	total, err := r.queries.CountProducts(ctx, sqlc.CountProductsParams{
		CategoryID: filter.CategoryID,
		Search:     filter.Search,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return port.ProductList{}, err
	}

	products := make([]model.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, productFromListRow(row))
	}
	return port.ProductList{
		Items: products,
		Total: int(total),
	}, nil
}

func (r *CatalogRepository) GetProduct(ctx context.Context, id string) (model.Product, error) {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.GetProduct")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.String("product_id", id),
	)
	row, err := r.queries.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Product{}, model.ErrProductNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Product{}, err
	}
	return productFromGetRow(row), nil
}

func (r *CatalogRepository) ValidateProducts(ctx context.Context, items []model.OrderItemInput) ([]model.ValidatedItem, int64, error) {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.ValidateProducts")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "SELECT"),
		attribute.Int("item_count", len(items)),
	)
	products, err := r.queries.GetProductsByIDs(ctx, productIDs(items))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}

	byID := make(map[string]sqlc.GetProductsByIDsRow, len(products))
	for _, product := range products {
		byID[product.ID] = product
	}

	validated := make([]model.ValidatedItem, 0, len(items))
	var total int64
	for _, item := range items {
		product, ok := byID[item.ProductID]
		if !ok {
			err := fmt.Errorf("%w: %s", model.ErrProductNotFound, item.ProductID)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, 0, err
		}
		lineTotal := int64(item.Quantity * float64(product.Price))
		total += lineTotal
		validated = append(validated, model.ValidatedItem{
			ProductID:   product.ID,
			ProductName: product.Name,
			Unit:        product.Unit,
			Quantity:    item.Quantity,
			UnitPrice:   product.Price,
			LineTotal:   lineTotal,
		})
	}
	return validated, total, nil
}

func (r *CatalogRepository) ReserveStock(ctx context.Context, command model.ReserveStockCommand) (model.StockReservation, error) {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.ReserveStock")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("order_id", command.OrderID),
		attribute.Int("item_count", len(command.Items)),
	)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.StockReservation{}, err
	}
	defer rollback(ctx, tx)

	q := r.queries.WithTx(tx)
	existing, err := q.GetReservationByIdempotencyKey(ctx, command.IdempotencyKey)
	if err == nil {
		return reservationFromIDRow(existing), tx.Commit(ctx)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.StockReservation{}, err
	}

	locked, err := q.LockInventoriesForProducts(ctx, productIDs(command.Items))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.StockReservation{}, err
	}
	available := make(map[string]float64, len(locked))
	for _, inventory := range locked {
		available[inventory.ProductID] = inventory.OnHandQty - inventory.ReservedQty
	}
	for _, item := range command.Items {
		if available[item.ProductID] < item.Quantity {
			err := fmt.Errorf("%w: %s", model.ErrInsufficientStock, item.ProductID)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return model.StockReservation{}, err
		}
	}

	created, err := q.CreateReservation(ctx, sqlc.CreateReservationParams{
		ID:             newID("res"),
		OrderID:        command.OrderID,
		Status:         model.ReservationStatusReserved,
		IdempotencyKey: command.IdempotencyKey,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.StockReservation{}, err
	}

	for _, item := range command.Items {
		if err := q.CreateReservationItem(ctx, sqlc.CreateReservationItemParams{
			ID:            newID("res_item"),
			ReservationID: created.ID,
			ProductID:     item.ProductID,
			Quantity:      item.Quantity,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return model.StockReservation{}, err
		}
		if err := q.AddReservedStock(ctx, sqlc.AddReservedStockParams{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return model.StockReservation{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.StockReservation{}, err
	}
	reservation := reservationFromCreateRow(created)
	reservation.Items = command.Items
	return reservation, nil
}

func (r *CatalogRepository) ReleaseStock(ctx context.Context, orderID string) (model.StockReservation, error) {
	return r.transitionReservation(ctx, orderID, model.ReservationStatusReleased)
}

func (r *CatalogRepository) CommitStock(ctx context.Context, orderID string) (model.StockReservation, error) {
	return r.transitionReservation(ctx, orderID, model.ReservationStatusCommitted)
}

func (r *CatalogRepository) transitionReservation(ctx context.Context, orderID, targetStatus string) (model.StockReservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.StockReservation{}, err
	}
	defer rollback(ctx, tx)

	q := r.queries.WithTx(tx)
	result, err := r.transitionReservationTx(ctx, q, orderID, targetStatus)
	if err != nil {
		return model.StockReservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.StockReservation{}, err
	}
	return result, nil
}

func (r *CatalogRepository) ProcessOrderEvent(ctx context.Context, event messaging.EventEnvelope) (bool, error) {
	ctx, span := catalogRepositoryTracer.Start(ctx, "postgres.CatalogRepository.ProcessOrderEvent")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", "transaction"),
		attribute.String("event_id", event.EventID),
		attribute.String("event_type", event.EventType),
		attribute.String("order_id", event.AggregateID),
	)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer rollback(ctx, tx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO inbox_events (event_id, event_type, aggregate_id, correlation_id, traceparent)
		VALUES ($1, $2, $3, $4, $5)
	`, event.EventID, event.EventType, event.AggregateID, event.CorrelationID, nil); err != nil {
		if isUniqueViolation(err) {
			return true, tx.Commit(ctx)
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	q := r.queries.WithTx(tx)
	switch event.EventType {
	case "OrderConfirmed":
		if _, err := r.transitionReservationTx(ctx, q, event.AggregateID, model.ReservationStatusCommitted); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	case "OrderCancelled":
		if _, err := r.transitionReservationTx(ctx, q, event.AggregateID, model.ReservationStatusReleased); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	return false, nil
}

func (r *CatalogRepository) transitionReservationTx(ctx context.Context, q *sqlc.Queries, orderID, targetStatus string) (model.StockReservation, error) {
	reservation, err := q.GetReservationByOrderID(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.StockReservation{}, model.ErrReservationNotFound
	}
	if err != nil {
		return model.StockReservation{}, err
	}
	if reservation.Status == targetStatus {
		result := reservationFromOrderRow(reservation)
		result.Items = reservationItemsToModelItems(ctx, q, reservation.ID)
		return result, nil
	}
	if reservation.Status != model.ReservationStatusReserved {
		result := reservationFromOrderRow(reservation)
		result.Items = reservationItemsToModelItems(ctx, q, reservation.ID)
		return result, nil
	}

	items, err := q.ListReservationItems(ctx, reservation.ID)
	if err != nil {
		return model.StockReservation{}, err
	}
	if _, err := q.LockInventoriesForProducts(ctx, reservationItemProductIDs(items)); err != nil {
		return model.StockReservation{}, err
	}

	for _, item := range items {
		if targetStatus == model.ReservationStatusCommitted {
			err = q.CommitReservedStock(ctx, sqlc.CommitReservedStockParams{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		} else {
			err = q.ReleaseReservedStock(ctx, sqlc.ReleaseReservedStockParams{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		}
		if err != nil {
			return model.StockReservation{}, err
		}
	}

	updated, err := q.UpdateReservationStatus(ctx, sqlc.UpdateReservationStatusParams{
		ID:     reservation.ID,
		Status: targetStatus,
	})
	if err != nil {
		return model.StockReservation{}, err
	}
	result := reservationFromUpdateRow(updated)
	result.Items = reservationRowsToModelItems(items)
	return result, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func productIDs(items []model.OrderItemInput) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProductID)
	}
	return ids
}

func reservationItemProductIDs(items []sqlc.ListReservationItemsRow) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProductID)
	}
	return ids
}

func reservationItemsToModelItems(ctx context.Context, q *sqlc.Queries, reservationID string) []model.OrderItemInput {
	items, err := q.ListReservationItems(ctx, reservationID)
	if err != nil {
		return nil
	}
	return reservationRowsToModelItems(items)
}

func reservationRowsToModelItems(items []sqlc.ListReservationItemsRow) []model.OrderItemInput {
	result := make([]model.OrderItemInput, 0, len(items))
	for _, item := range items {
		result = append(result, model.OrderItemInput{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}
	return result
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_local"
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func productFromListRow(row sqlc.ListProductsRow) model.Product {
	return model.Product{
		ID:            row.ID,
		CategoryID:    row.CategoryID,
		CategoryName:  row.CategoryName,
		SKU:           row.Sku,
		Name:          row.Name,
		Brand:         row.Brand,
		Unit:          row.Unit,
		Price:         row.Price,
		WeightKG:      row.WeightKg,
		RequiresTruck: row.RequiresTruck,
		AvailableQty:  row.AvailableQty,
	}
}

func productFromGetRow(row sqlc.GetProductRow) model.Product {
	return model.Product{
		ID:            row.ID,
		CategoryID:    row.CategoryID,
		CategoryName:  row.CategoryName,
		SKU:           row.Sku,
		Name:          row.Name,
		Brand:         row.Brand,
		Unit:          row.Unit,
		Price:         row.Price,
		WeightKG:      row.WeightKg,
		RequiresTruck: row.RequiresTruck,
		AvailableQty:  row.AvailableQty,
	}
}

func reservationFromIDRow(row sqlc.GetReservationByIdempotencyKeyRow) model.StockReservation {
	return model.StockReservation{
		ID:             row.ID,
		OrderID:        row.OrderID,
		Status:         row.Status,
		IdempotencyKey: row.IdempotencyKey,
	}
}

func reservationFromOrderRow(row sqlc.GetReservationByOrderIDRow) model.StockReservation {
	return model.StockReservation{
		ID:             row.ID,
		OrderID:        row.OrderID,
		Status:         row.Status,
		IdempotencyKey: row.IdempotencyKey,
	}
}

func reservationFromCreateRow(row sqlc.CreateReservationRow) model.StockReservation {
	return model.StockReservation{
		ID:             row.ID,
		OrderID:        row.OrderID,
		Status:         row.Status,
		IdempotencyKey: row.IdempotencyKey,
	}
}

func reservationFromUpdateRow(row sqlc.UpdateReservationStatusRow) model.StockReservation {
	return model.StockReservation{
		ID:             row.ID,
		OrderID:        row.OrderID,
		Status:         row.Status,
		IdempotencyKey: row.IdempotencyKey,
	}
}
