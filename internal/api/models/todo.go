package models

import "time"

type Todo struct {
	Id string `json:"id,omitempty" db:"id"`
	UserId *string `json:"user_id,omitempty" db:"user_id"`
	Title string `json:"title" db:"title"`
	Content string `json:"content" db:"content"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}

func (Todo) TableName() string {
	return "todo" 
}
