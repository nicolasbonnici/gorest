package processor

import (
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
)

// ProcessorConfig holds all configuration for a Processor instance.
type ProcessorConfig[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	// Required fields
	DB        database.Database
	CRUD      *crud.CRUD[TModel]
	Converter ModelConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]

	// Pagination settings
	PaginationLimit    int
	PaginationMaxLimit int

	// Filtering and ordering
	AllowedFields []string          // Fields allowed for filtering/ordering
	FieldMap      map[string]string // Optional JSON->DB field mapping

	// Context enrichment (auth, multi-tenancy)
	ContextEnrichers []ContextEnricher

	// Error handling
	ErrorHandler ErrorHandler

	// Validation functions
	ValidateCreate func(TCreateDTO) error
	ValidateUpdate func(TUpdateDTO) error
}

// New creates a new Processor with the given configuration.
// It applies sensible defaults for optional fields.
func New[TModel crud.Model, TCreateDTO any, TUpdateDTO any, TResponseDTO any](
	config ProcessorConfig[TModel, TCreateDTO, TUpdateDTO, TResponseDTO],
) Processor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO] {
	// Apply defaults
	if config.PaginationLimit == 0 {
		config.PaginationLimit = 30
	}
	if config.PaginationMaxLimit == 0 {
		config.PaginationMaxLimit = 100
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = &DefaultErrorHandler{}
	}

	return &StandardProcessor[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]{
		config: config,
	}
}
