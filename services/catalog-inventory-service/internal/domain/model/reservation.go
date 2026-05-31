package model

const (
	ReservationStatusReserved  = "RESERVED"
	ReservationStatusCommitted = "COMMITTED"
	ReservationStatusReleased  = "RELEASED"
	ReservationStatusFailed    = "FAILED"
)

type StockReservation struct {
	ID             string
	OrderID        string
	Status         string
	IdempotencyKey string
	FailureReason  string
	Items          []OrderItemInput
}

type ReserveStockCommand struct {
	OrderID        string
	IdempotencyKey string
	Items          []OrderItemInput
}
