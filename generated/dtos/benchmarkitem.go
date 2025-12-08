package dtos

import "time"

type BenchmarkItemDTO struct {
	Id string `json:"id"`
	Name *string `json:"name"`
	Value *int `json:"value"`
	Description *string `json:"description"`
	CreatedAt *time.Time `json:"createdAt"`
}

type BenchmarkItemCreateDTO struct {
	Name *string `json:"name"`
	Value *int `json:"value"`
	Description *string `json:"description"`
}

type BenchmarkItemUpdateDTO struct {
	Name *string `json:"name"`
	Value *int `json:"value"`
	Description *string `json:"description"`
}
