package processor

import (
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/filter"
)

type ProcessorConfig[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	DB        database.Database
	CRUD      *crud.CRUD[TModel]
	Converter ModelConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]

	PaginationLimit    int
	PaginationMaxLimit int
	// CountMode defaults to the application-wide DefaultCountMode. Clients opt
	// out per request with ?count=false regardless of this setting.
	CountMode crud.CountMode

	AllowedFields []string
	FieldMap      map[string]string

	ContextEnrichers []ContextEnricher
	ErrorHandler     ErrorHandler

	ValidateCreate func(TCreateDTO) error
	ValidateUpdate func(TUpdateDTO) error
}

func New[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any](
	config ProcessorConfig[TModel, TCreateDTO, TUpdateDTO, TResponseDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	if config.PaginationLimit == 0 {
		config.PaginationLimit = 30
	}
	if config.PaginationMaxLimit == 0 {
		config.PaginationMaxLimit = 100
	}
	if config.CountMode == "" {
		config.CountMode = DefaultCountMode()
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = &DefaultErrorHandler{}
	}

	return &StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]{
		config: config,
		// Built once here rather than twice (filters + ordering) per request.
		allowedSet: filter.AllowedSet(config.AllowedFields),
	}
}
