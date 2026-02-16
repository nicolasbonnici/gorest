package dtos

import "time"

type UserDTO struct {
	Id        string     `json:"id"`
	Firstname string     `json:"firstname"`
	Lastname  string     `json:"lastname"`
	Email     string     `json:"email"`
	Password  *string    `json:"password"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt *time.Time `json:"createdAt"`
}

type UserCreateDTO struct {
	Firstname string  `json:"firstname"`
	Lastname  string  `json:"lastname"`
	Email     string  `json:"email"`
	Password  *string `json:"password"`
}

type UserUpdateDTO struct {
	Firstname string  `json:"firstname"`
	Lastname  string  `json:"lastname"`
	Email     string  `json:"email"`
	Password  *string `json:"password"`
}
