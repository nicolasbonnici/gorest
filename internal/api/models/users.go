package models

import "time"

type Users struct {
	Id string `json:"id,omitempty" db:"id"`
	Firstname string `json:"firstname" db:"firstname"`
	Lastname string `json:"lastname" db:"lastname"`
	Email string `json:"email" db:"email"`
	Password *string `json:"password,omitempty" db:"password"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}

func (Users) TableName() string {
	return "users" 
}
