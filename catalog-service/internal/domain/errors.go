package domain

import "errors"

var (
	// ErrProductNotFound возвращается, когда товар не найден в базе.
	ErrProductNotFound = errors.New("product not found")
)
