package models

import "time"

type BenchmarkItem struct {
	Id string `json:"id,omitempty" db:"id"`
	Name *string `json:"name,omitempty" db:"name"`
	Value *int `json:"value,omitempty" db:"value"`
	Description *string `json:"description,omitempty" db:"description"`
	CreatedAt *time.Time `json:"createdAt,omitempty" db:"created_at"`
}

func (BenchmarkItem) TableName() string {
	return "benchmark_items" 
}
