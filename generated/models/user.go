package models

import "time"

type User struct {
	Id        string     `json:"id,omitempty" db:"id"`
	Firstname string     `json:"firstname" db:"firstname"`
	Lastname  string     `json:"lastname" db:"lastname"`
	Email     string     `json:"email" db:"email"`
	Password  *string    `json:"password,omitempty" db:"password"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
	CreatedAt *time.Time `json:"createdAt,omitempty" db:"created_at"`
}

func (User) TableName() string {
	return "users"
}
