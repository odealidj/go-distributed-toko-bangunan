package model

import "time"

const (
	OrderStatusPending        = "PENDING"
	OrderStatusStockReserved  = "STOCK_RESERVED"
	OrderStatusWaitingPayment = "WAITING_PAYMENT"
	OrderStatusConfirmed      = "CONFIRMED"
	OrderStatusCancelled      = "CANCELLED"
	OrderStatusRejected       = "REJECTED"
)

const (
	SagaStatusStarted            = "STARTED"
	SagaStatusStockReserved      = "STOCK_RESERVED"
	SagaStatusPaymentCreated     = "PAYMENT_CREATED"
	SagaStatusCompleted          = "COMPLETED"
	SagaStatusCompensated        = "COMPENSATED"
	SagaStatusFailedManualRepair = "FAILED_REQUIRES_MANUAL_REPAIR"
)

const (
	SagaStepProductValidated = "product_validated"
	SagaStepReserveStock     = "reserve_stock"
	SagaStepCreatePayment    = "create_payment"
	SagaStepCommitStock      = "commit_stock"
	SagaStepReleaseStock     = "release_stock"
)

const (
	SagaStepStatusSucceeded = "SUCCEEDED"
	SagaStepStatusFailed    = "FAILED"
)

const (
	OutboxStatusPending = "PENDING"
)

const (
	PaymentModeSuccess = "SUCCESS"
	PaymentModeFailure = "FAILURE"
	PaymentModeManual  = "MANUAL"
)

const (
	PaymentStatusPending   = "PENDING"
	PaymentStatusSucceeded = "SUCCEEDED"
	PaymentStatusFailed    = "FAILED"
)

type Order struct {
	ID              string
	CustomerName    string
	CustomerPhone   string
	CustomerAddress string
	Status          string
	TotalAmount     int64
	PaymentID       string
	CorrelationID   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Items           []OrderItem
}

type OrderItem struct {
	ID          string
	OrderID     string
	ProductID   string
	ProductName string
	Unit        string
	Quantity    float64
	UnitPrice   int64
	LineTotal   int64
}

type OrderItemInput struct {
	ProductID string
	Quantity  float64
}

type ValidatedOrderItem struct {
	ProductID   string
	ProductName string
	Unit        string
	Quantity    float64
	UnitPrice   int64
	LineTotal   int64
}

type CreateCheckoutCommand struct {
	CustomerName    string
	CustomerPhone   string
	CustomerAddress string
	PaymentMode     string
	CorrelationID   string
	CausationID     string
	Items           []OrderItemInput
}

type Payment struct {
	ID     string
	Status string
}

type SagaInstance struct {
	ID            string
	OrderID       string
	Status        string
	CurrentStep   string
	CorrelationID string
}

type SagaStep struct {
	ID             string
	SagaID         string
	StepName       string
	Status         string
	IdempotencyKey string
	ErrorMessage   string
}

type OutboxEvent struct {
	ID            string
	AggregateID   string
	AggregateType string
	EventType     string
	CorrelationID string
	CausationID   string
	Traceparent   string
	Payload       []byte
	Status        string
	CreatedAt     time.Time
}

type SagaTransition struct {
	OrderID         string
	OrderStatus     string
	PaymentID       string
	SagaID          string
	SagaStatus      string
	SagaCurrentStep string
	CompleteSaga    bool
	Step            SagaStep
	Event           OutboxEvent
}
