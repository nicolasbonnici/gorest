package dtos

import (
	"time"

	"github.com/google/uuid"
)

type UserCreateDTO struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type UserUpdateDTO struct {
	Email     *string `json:"email,omitempty"`
	Password  *string `json:"password,omitempty"`
	Firstname *string `json:"firstname,omitempty"`
	Lastname  *string `json:"lastname,omitempty"`
}

type UserResponseDTO struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Firstname string     `json:"firstname"`
	Lastname  string     `json:"lastname"`
	Roles     []string   `json:"roles,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
