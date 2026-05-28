package model

const (
	PaymentStatusPending   = "PENDING"
	PaymentStatusSucceeded = "SUCCEEDED"
	PaymentStatusFailed    = "FAILED"
	PaymentStatusCancelled = "CANCELLED"
)

const (
	PaymentModeSuccess = "SUCCESS"
	PaymentModeFailure = "FAILURE"
	PaymentModeManual  = "MANUAL"
)

type Payment struct {
	ID             string
	OrderID        string
	Amount         int64
	Status         string
	PaymentMode    string
	IdempotencyKey string
}

type PaymentAttempt struct {
	ID        string
	PaymentID string
	Status    string
	Reason    string
}

type CreatePaymentCommand struct {
	OrderID        string
	Amount         int64
	PaymentMode    string
	IdempotencyKey string
}

type CancelPaymentCommand struct {
	PaymentID string
	OrderID   string
	Reason    string
}
