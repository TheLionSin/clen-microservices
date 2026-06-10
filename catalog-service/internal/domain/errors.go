package domain

import "errors"

var (
	// ErrProductNotFound возвращается, когда товар не найден в базе.
	ErrProductNotFound  = errors.New("product not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryInUse    = errors.New("cannot delete category because it contains products")
)
