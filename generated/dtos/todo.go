package dtos

import "time"

type TodoDTO struct {
	Id        string     `json:"id"`
	UserId    *string    `json:"userId"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt *time.Time `json:"createdAt"`
}

type TodoCreateDTO struct {
	UserId  *string `json:"userId"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
}

type TodoUpdateDTO struct {
	UserId  *string `json:"userId"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
}
