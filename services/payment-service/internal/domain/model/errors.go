package model

import "errors"

var (
	ErrInvalidInput    = errors.New("input tidak valid")
	ErrPaymentNotFound = errors.New("payment tidak ditemukan")
	ErrPaymentConflict = errors.New("payment conflict")
)
