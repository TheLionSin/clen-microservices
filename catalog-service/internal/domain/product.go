package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Name     string
}
type Product struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Description string
	Price       int64
	Stock       int
	ImageURLs   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
