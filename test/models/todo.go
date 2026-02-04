package models

import (
	"time"

	"github.com/google/uuid"
)

type Todo struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    *uuid.UUID `json:"userId,omitempty" db:"user_id"`
	Title     string     `json:"title" db:"title"`
	Content   string     `json:"content" db:"content"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}

func (Todo) TableName() string {
	return "todo"
}
