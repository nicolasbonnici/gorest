package models

import (
	"time"

	"github.com/google/uuid"
)

type BenchmarkItem struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        *string    `json:"name,omitempty" db:"name"`
	Value       *int       `json:"value,omitempty" db:"value"`
	Description *string    `json:"description,omitempty" db:"description"`
	CreatedAt   *time.Time `json:"createdAt,omitempty" db:"created_at"`
}

func (BenchmarkItem) TableName() string {
	return "benchmark_items"
}
