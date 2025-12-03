package plugin

import (
	"github.com/gofiber/fiber/v2"
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
	// Name returns the command name (e.g., "models", "resources")
	Name() string

	// Description returns a brief description of what the command does
	Description() string

	// Run executes the command and returns a result
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
