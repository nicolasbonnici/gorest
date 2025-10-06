package models

import "time"

type Model struct {
	ID        interface{} `json:"id" db:"id"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time  `json:"updated_at" db:"updated_at"`
}

type Model interface {
	TableName() string
}