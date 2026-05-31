package plugin

import (
	"github.com/gofiber/fiber/v3"
)

const (
	ConfigKeyDependencies = "__dependencies"
	ConfigKeyVersion      = "__version"
)

type Plugin interface {
	Name() string

	Initialize(config map[string]any) error

	Handler() fiber.Handler
}

type EndpointSetup interface {
	SetupEndpoints(router fiber.Router) error
}

type Command interface {
	Name() string
	Description() string
	Run(ctx *CommandContext) *CommandResult
}

type CommandContext struct {
	Config any

	Args []string

	ProgressCallback func(message string)
}

type CommandResult struct {
	Success bool

	Error error

	FilesCreated []string

	FilesModified []string

	Message string
}

type CommandProvider interface {
	Commands() []Command
}

type MigrationProvider interface {
	// MigrationSource returns a migration source for this plugin
	// The source name should match the plugin name
	// Returns any to avoid circular dependency with migrations package
	// Actual type should be migrations.MigrationSource
	MigrationSource() any

	// MigrationDependencies returns list of sources this plugin depends on
	// Example: ["app", "users-plugin"]
	// Ensures migrations run in correct order
	// Return nil or empty slice if no dependencies
	MigrationDependencies() []string
}

// PluginDependencies is an optional interface for plugins that depend on other plugins.
// Dependencies must be declared by their plugin name (matching Plugin.Name()).
// The plugin loader ensures:
//   - All dependencies are registered and enabled
//   - Dependencies are initialized before dependent plugins
//   - Circular dependencies are detected and rejected
//   - Dependencies are injected via Initialize() config under ConfigKeyDependencies
type PluginDependencies interface {
	Dependencies() []string
}

type OpenAPIProvider interface {
	GetOpenAPIResources() []OpenAPIResource
}

type OpenAPIResource struct {
	Name          string
	PluralName    string
	BasePath      string
	Tags          []string
	ResponseModel any
	CreateModel   any
	UpdateModel   any
	Description   string
}
