package domain

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       int64
	Stock       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
