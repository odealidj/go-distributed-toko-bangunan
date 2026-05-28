package model

import "errors"

var (
	ErrInvalidInput        = errors.New("input tidak valid")
	ErrProductNotFound     = errors.New("produk tidak ditemukan")
	ErrInsufficientStock   = errors.New("stock tidak mencukupi")
	ErrReservationNotFound = errors.New("reservasi stock tidak ditemukan")
)
