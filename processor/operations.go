package processor

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/query"
)

type CreateHookFunc[TModel any, TCreateDTO any] func(c fiber.Ctx, dto TCreateDTO, model *TModel) error

type UpdateHookFunc[TModel any, TUpdateDTO any] func(c fiber.Ctx, dto TUpdateDTO, model *TModel) error

type DeleteHookFunc func(c fiber.Ctx, id any) error

type GetByIDHookFunc[TModel any] func(c fiber.Ctx, id any) error

type GetAllHookFunc[TModel any] func(c fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error
