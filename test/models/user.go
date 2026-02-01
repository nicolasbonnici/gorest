package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Firstname string     `json:"firstname" db:"firstname"`
	Lastname  string     `json:"lastname" db:"lastname"`
	Email     string     `json:"email" db:"email"`
	Password  *string    `json:"password,omitempty" db:"password"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}

func (User) TableName() string {
	return "users"
}
