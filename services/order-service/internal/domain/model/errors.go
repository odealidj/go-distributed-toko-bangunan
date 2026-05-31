package model

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrOrderNotFound     = errors.New("order not found")
	ErrOrderConflict     = errors.New("order conflict")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrPaymentFailed     = errors.New("payment failed")
)
