package processor

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/query"
)

// CreateHookFunc is called before CRUD.Create() operation.
// It receives the Fiber context, the parsed CreateDTO, and a pointer to the Model.
// It can modify the model or return an error to abort the operation.
type CreateHookFunc[TModel any, TCreateDTO any] func(c *fiber.Ctx, dto TCreateDTO, model *TModel) error

// UpdateHookFunc is called before CRUD.Update() operation.
// It receives the Fiber context, the parsed UpdateDTO, and a pointer to the Model.
// It can modify the model or return an error to abort the operation.
type UpdateHookFunc[TModel any, TUpdateDTO any] func(c *fiber.Ctx, dto TUpdateDTO, model *TModel) error

// DeleteHookFunc is called before CRUD.Delete() operation.
// It receives the Fiber context and the ID being deleted.
// It can return an error to abort the operation.
type DeleteHookFunc func(c *fiber.Ctx, id any) error

// GetByIDHookFunc is called before CRUD.GetByID() operation.
// It receives the Fiber context and the ID being fetched.
// It can return an error to abort the operation.
type GetByIDHookFunc[TModel any] func(c *fiber.Ctx, id any) error

// GetAllHookFunc is called before CRUD.GetAllPaginated() operation.
// It receives the Fiber context and pointers to the conditions and orderBy slices.
// It can modify the query conditions or ordering, or return an error to abort the operation.
type GetAllHookFunc[TModel any] func(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error
