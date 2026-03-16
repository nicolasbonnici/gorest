package processor

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	auth "github.com/nicolasbonnici/gorest-auth"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/filter"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/pagination"
	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/response"
)

type Processor[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	GetAll(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error

	WithCreateHook(hook CreateHookFunc[TModel, TCreateDTO]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithUpdateHook(hook UpdateHookFunc[TModel, TUpdateDTO]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithDeleteHook(hook DeleteHookFunc) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithGetByIDHook(hook GetByIDHookFunc[TModel]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithGetAllHook(hook GetAllHookFunc[TModel]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
}

type StandardProcessor[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	config ProcessorConfig[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]

	createHook  CreateHookFunc[TModel, TCreateDTO]
	updateHook  UpdateHookFunc[TModel, TUpdateDTO]
	deleteHook  DeleteHookFunc
	getByIDHook GetByIDHookFunc[TModel]
	getAllHook  GetAllHookFunc[TModel]
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Create(c *fiber.Ctx) error {
	var createDTO TCreateDTO
	if err := c.BodyParser(&createDTO); err != nil {
		logger.Log.Error("Failed to parse request body", "error", err, "path", c.Path())
		return p.config.ErrorHandler.HandleError(c, err, "parse")
	}

	if p.config.ValidateCreate != nil {
		if err := p.config.ValidateCreate(createDTO); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "validate")
		}
	}

	model := p.config.Converter.CreateDTOToModel(createDTO)

	for _, enricher := range p.config.ContextEnrichers {
		if err := enricher(c, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "enrich")
		}
	}

	if p.createHook != nil {
		if err := p.createHook(c, createDTO, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	ctx := auth.Context(c)
	if err := p.config.CRUD.Create(ctx, model); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "create")
	}

	id, err := getModelID(&model)
	if err != nil {
		dto := p.config.Converter.ModelToResponseDTO(model)
		return response.SendFormatted(c, fiber.StatusCreated, dto)
	}

	created, err := p.config.CRUD.GetByID(ctx, id)
	if err != nil {
		dto := p.config.Converter.ModelToResponseDTO(model)
		return response.SendFormatted(c, fiber.StatusCreated, dto)
	}

	dto := p.config.Converter.ModelToResponseDTO(*created)
	return response.SendFormatted(c, fiber.StatusCreated, dto)
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	if p.getByIDHook != nil {
		if err := p.getByIDHook(c, id); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	ctx := auth.Context(c)
	item, err := p.config.CRUD.GetByID(ctx, id)
	if err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "getById")
	}

	dto := p.config.Converter.ModelToResponseDTO(*item)
	return response.SendFormatted(c, fiber.StatusOK, dto)
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) GetAll(c *fiber.Ctx) error {
	limit := pagination.ParseIntQuery(c, "limit", p.config.PaginationLimit, p.config.PaginationMaxLimit)
	page := pagination.ParseIntQuery(c, "page", 1, 10000)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	includeCount := c.Query("count", "true") != "false"

	queryParams := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams.Add(string(key), string(value))
	})

	var conditions []query.Condition
	if len(p.config.AllowedFields) > 0 || p.config.FieldMap != nil {
		var filters *filter.FilterSet
		if p.config.FieldMap != nil {
			filters = filter.NewFilterSetWithMapping(p.config.FieldMap, p.config.DB.Dialect())
		} else {
			filters = filter.NewFilterSet(p.config.AllowedFields, p.config.DB.Dialect())
		}

		if err := filters.ParseFromQuery(queryParams); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "parseFilters")
		}
		conditions = filters.Conditions()
	}

	var orderBy []crud.OrderByClause
	if len(p.config.AllowedFields) > 0 || p.config.FieldMap != nil {
		var ordering *filter.OrderSet
		if p.config.FieldMap != nil {
			ordering = filter.NewOrderSetWithMapping(p.config.FieldMap)
		} else {
			ordering = filter.NewOrderSet(p.config.AllowedFields)
		}

		if err := ordering.ParseFromQuery(queryParams); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "parseOrdering")
		}

		orderClauses := ordering.OrderClauses()
		orderBy = make([]crud.OrderByClause, len(orderClauses))
		for i, oc := range orderClauses {
			orderBy[i] = crud.OrderByClause{
				Column:    oc.Column,
				Direction: oc.Direction,
			}
		}
	}

	if p.getAllHook != nil {
		if err := p.getAllHook(c, &conditions, &orderBy); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	ctx := auth.Context(c)
	result, err := p.config.CRUD.GetAllPaginated(ctx, crud.PaginationOptions{
		Limit:        limit,
		Offset:       offset,
		IncludeCount: includeCount,
		Conditions:   conditions,
		OrderBy:      orderBy,
	})
	if err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "getAll")
	}

	dtoItems := p.config.Converter.ModelsToResponseDTOs(result.Items)
	return pagination.SendHydraCollection(c, dtoItems, result.Total, limit, page, p.config.PaginationLimit)
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var updateDTO TUpdateDTO
	if err := c.BodyParser(&updateDTO); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "parse")
	}

	if p.config.ValidateUpdate != nil {
		if err := p.config.ValidateUpdate(updateDTO); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "validate")
		}
	}

	model := p.config.Converter.UpdateDTOToModel(updateDTO)

	for _, enricher := range p.config.ContextEnrichers {
		if err := enricher(c, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "enrich")
		}
	}

	if p.updateHook != nil {
		if err := p.updateHook(c, updateDTO, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	ctx := auth.Context(c)
	if err := p.config.CRUD.Update(ctx, id, model); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "update")
	}

	dto := p.config.Converter.ModelToResponseDTO(model)
	return response.SendFormatted(c, fiber.StatusOK, dto)
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if p.deleteHook != nil {
		if err := p.deleteHook(c, id); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	ctx := auth.Context(c)
	if err := p.config.CRUD.Delete(ctx, id); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "delete")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithCreateHook(
	hook CreateHookFunc[TModel, TCreateDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.createHook = hook
	return p
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithUpdateHook(
	hook UpdateHookFunc[TModel, TUpdateDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.updateHook = hook
	return p
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithDeleteHook(
	hook DeleteHookFunc,
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.deleteHook = hook
	return p
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithGetByIDHook(
	hook GetByIDHookFunc[TModel],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.getByIDHook = hook
	return p
}

func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithGetAllHook(
	hook GetAllHookFunc[TModel],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.getAllHook = hook
	return p
}
