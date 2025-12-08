package models

import "time"

type Todo struct {
	Id string `json:"id,omitempty" db:"id"`
	UserId *string `json:"userId,omitempty" db:"user_id"`
	Title string `json:"title" db:"title"`
	Content string `json:"content" db:"content"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
	CreatedAt *time.Time `json:"createdAt,omitempty" db:"created_at"`
}

func (Todo) TableName() string {
	return "todo" 
}
