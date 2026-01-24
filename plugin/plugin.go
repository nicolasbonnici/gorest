package plugin

import (
	"github.com/gofiber/fiber/v2"
)

const (
	ConfigKeyDependencies = "__dependencies"
	ConfigKeyVersion      = "__version"
)

type Plugin interface {
	Name() string

	Initialize(config map[string]interface{}) error

	Handler() fiber.Handler
}

type EndpointSetup interface {
	SetupEndpoints(app *fiber.App) error
}

type Command interface {
	Name() string
	Description() string
	Run(ctx *CommandContext) *CommandResult
}

type CommandContext struct {
	Config interface{}

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
	// Returns interface{} to avoid circular dependency with migrations package
	// Actual type should be migrations.MigrationSource
	MigrationSource() interface{}

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
	ResponseModel interface{}
	CreateModel   interface{}
	UpdateModel   interface{}
	Description   string
}
