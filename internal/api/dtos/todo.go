package dtos

import "time"

type TodoDTO struct {
	Id string `json:"id"`
	UserId *string `json:"user_id"`
	Title string `json:"title"`
	Content string `json:"content"`
	UpdatedAt *time.Time `json:"updated_at"`
	CreatedAt *time.Time `json:"created_at"`
}

type TodoCreateDTO struct {
	UserId *string `json:"user_id"`
	Title string `json:"title"`
	Content string `json:"content"`
}

type TodoUpdateDTO struct {
	UserId *string `json:"user_id"`
	Title string `json:"title"`
	Content string `json:"content"`
}
