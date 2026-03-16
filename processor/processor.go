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

// Processor defines the interface for a unified API processor that handles
// all CRUD operations with consistent behavior across endpoints.
type Processor[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] interface {
	// Create handles POST /resources
	Create(c *fiber.Ctx) error

	// GetByID handles GET /resources/:id
	GetByID(c *fiber.Ctx) error

	// GetAll handles GET /resources (paginated, filterable, orderable)
	GetAll(c *fiber.Ctx) error

	// Update handles PUT /resources/:id
	Update(c *fiber.Ctx) error

	// Delete handles DELETE /resources/:id
	Delete(c *fiber.Ctx) error

	// Customization methods (fluent API)
	WithCreateHook(hook CreateHookFunc[TModel, TCreateDTO]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithUpdateHook(hook UpdateHookFunc[TModel, TUpdateDTO]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithDeleteHook(hook DeleteHookFunc) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithGetByIDHook(hook GetByIDHookFunc[TModel]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
	WithGetAllHook(hook GetAllHookFunc[TModel]) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]
}

// StandardProcessor is the default implementation of Processor.
type StandardProcessor[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	config ProcessorConfig[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]

	// Custom hooks
	createHook  CreateHookFunc[TModel, TCreateDTO]
	updateHook  UpdateHookFunc[TModel, TUpdateDTO]
	deleteHook  DeleteHookFunc
	getByIDHook GetByIDHookFunc[TModel]
	getAllHook  GetAllHookFunc[TModel]
}

// Create handles POST /resources
// Flow: Parse DTO → Validate → Convert to Model → Enrich → Custom Hook → CRUD.Create() → Fetch created → Convert to DTO → Send 201
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Create(c *fiber.Ctx) error {
	// Parse CreateDTO from request body
	var createDTO TCreateDTO
	if err := c.BodyParser(&createDTO); err != nil {
		logger.Log.Error("Failed to parse request body", "error", err, "path", c.Path())
		return p.config.ErrorHandler.HandleError(c, err, "parse")
	}

	// Validate DTO (if validation function provided)
	if p.config.ValidateCreate != nil {
		if err := p.config.ValidateCreate(createDTO); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "validate")
		}
	}

	// Convert DTO to Model
	model := p.config.Converter.CreateDTOToModel(createDTO)

	// Apply context enrichers (user_id, tenant_id, etc.)
	for _, enricher := range p.config.ContextEnrichers {
		if err := enricher(c, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "enrich")
		}
	}

	// Run custom CreateHook (if provided)
	if p.createHook != nil {
		if err := p.createHook(c, createDTO, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	// Execute CRUD.Create() - all hook layers execute (StateProcessor, BeforeQuery, AfterQuery, Serializer)
	ctx := auth.Context(c)
	if err := p.config.CRUD.Create(ctx, model); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "create")
	}

	// Fetch the created entity with GetByID to ensure we have the full entity with generated fields
	id, err := getModelID(&model)
	if err != nil {
		// If we can't get the ID, return the model as-is
		dto := p.config.Converter.ModelToResponseDTO(model)
		return response.SendFormatted(c, fiber.StatusCreated, dto)
	}

	created, err := p.config.CRUD.GetByID(ctx, id)
	if err != nil {
		// If fetch fails, return the model as-is
		dto := p.config.Converter.ModelToResponseDTO(model)
		return response.SendFormatted(c, fiber.StatusCreated, dto)
	}

	// Convert to ResponseDTO
	dto := p.config.Converter.ModelToResponseDTO(*created)

	// Send 201 Created with formatted response
	return response.SendFormatted(c, fiber.StatusCreated, dto)
}

// GetByID handles GET /resources/:id
// Flow: Extract ID → Custom Hook → CRUD.GetByID() → Convert to DTO → Send 200
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) GetByID(c *fiber.Ctx) error {
	// Extract ID from path params
	id := c.Params("id")

	// Run custom GetByIDHook (if provided)
	if p.getByIDHook != nil {
		if err := p.getByIDHook(c, id); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	// Execute CRUD.GetByID() - hooks execute (ModifySelectQuery, BeforeQuery, AfterQuery, SerializeOne)
	ctx := auth.Context(c)
	item, err := p.config.CRUD.GetByID(ctx, id)
	if err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "getById")
	}

	// Convert to ResponseDTO
	dto := p.config.Converter.ModelToResponseDTO(*item)

	// Send 200 OK with formatted response
	return response.SendFormatted(c, fiber.StatusOK, dto)
}

// GetAll handles GET /resources (paginated, filterable, orderable)
// Flow: Parse pagination/filters/ordering → Custom Hook → CRUD.GetAllPaginated() → Convert to DTOs → Send paginated response
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) GetAll(c *fiber.Ctx) error {
	// Parse pagination params
	limit := pagination.ParseIntQuery(c, "limit", p.config.PaginationLimit, p.config.PaginationMaxLimit)
	page := pagination.ParseIntQuery(c, "page", 1, 10000)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	includeCount := c.Query("count", "true") != "false"

	// Parse query params
	queryParams := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams.Add(string(key), string(value))
	})

	// Parse filters into conditions
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

	// Parse ordering into OrderBy clauses
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

	// Run custom GetAllHook (if provided)
	if p.getAllHook != nil {
		if err := p.getAllHook(c, &conditions, &orderBy); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	// Execute CRUD.GetAllPaginated() - hooks execute
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

	// Convert models to ResponseDTOs
	dtoItems := p.config.Converter.ModelsToResponseDTOs(result.Items)

	// Send paginated response using Hydra collection
	return pagination.SendHydraCollection(c, dtoItems, result.Total, limit, page, p.config.PaginationLimit)
}

// Update handles PUT /resources/:id
// Flow: Extract ID → Parse DTO → Validate → Convert to Model → Enrich → Custom Hook → CRUD.Update() → Convert to DTO → Send 200
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Update(c *fiber.Ctx) error {
	// Extract ID from path params
	id := c.Params("id")

	// Parse UpdateDTO from request body
	var updateDTO TUpdateDTO
	if err := c.BodyParser(&updateDTO); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "parse")
	}

	// Validate DTO (if validation function provided)
	if p.config.ValidateUpdate != nil {
		if err := p.config.ValidateUpdate(updateDTO); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "validate")
		}
	}

	// Convert DTO to Model
	model := p.config.Converter.UpdateDTOToModel(updateDTO)

	// Apply context enrichers (user_id, tenant_id, etc.)
	for _, enricher := range p.config.ContextEnrichers {
		if err := enricher(c, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "enrich")
		}
	}

	// Run custom UpdateHook (if provided)
	if p.updateHook != nil {
		if err := p.updateHook(c, updateDTO, &model); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	// Execute CRUD.Update() - all hook layers execute
	ctx := auth.Context(c)
	if err := p.config.CRUD.Update(ctx, id, model); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "update")
	}

	// Convert to ResponseDTO
	dto := p.config.Converter.ModelToResponseDTO(model)

	// Send 200 OK with formatted response
	return response.SendFormatted(c, fiber.StatusOK, dto)
}

// Delete handles DELETE /resources/:id
// Flow: Extract ID → Custom Hook → CRUD.Delete() → Send 204
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) Delete(c *fiber.Ctx) error {
	// Extract ID from path params
	id := c.Params("id")

	// Run custom DeleteHook (if provided)
	if p.deleteHook != nil {
		if err := p.deleteHook(c, id); err != nil {
			return p.config.ErrorHandler.HandleError(c, err, "hook")
		}
	}

	// Execute CRUD.Delete() - hooks execute (StateProcessor, ModifyDeleteQuery, BeforeQuery, AfterQuery)
	ctx := auth.Context(c)
	if err := p.config.CRUD.Delete(ctx, id); err != nil {
		return p.config.ErrorHandler.HandleError(c, err, "delete")
	}

	// Send 204 No Content
	return c.SendStatus(fiber.StatusNoContent)
}

// WithCreateHook registers a custom hook to run before CRUD.Create()
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithCreateHook(
	hook CreateHookFunc[TModel, TCreateDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.createHook = hook
	return p
}

// WithUpdateHook registers a custom hook to run before CRUD.Update()
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithUpdateHook(
	hook UpdateHookFunc[TModel, TUpdateDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.updateHook = hook
	return p
}

// WithDeleteHook registers a custom hook to run before CRUD.Delete()
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithDeleteHook(
	hook DeleteHookFunc,
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.deleteHook = hook
	return p
}

// WithGetByIDHook registers a custom hook to run before CRUD.GetByID()
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithGetByIDHook(
	hook GetByIDHookFunc[TModel],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.getByIDHook = hook
	return p
}

// WithGetAllHook registers a custom hook to run before CRUD.GetAllPaginated()
func (p *StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) WithGetAllHook(
	hook GetAllHookFunc[TModel],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	p.getAllHook = hook
	return p
}
