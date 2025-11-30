package plugin

import (
	"github.com/gofiber/fiber/v2"
)

type Plugin interface {
	Name() string

	Initialize(config map[string]interface{}) error

	Handler() fiber.Handler
}

// EndpointSetup is an optional interface for plugins that need to register endpoints.
type EndpointSetup interface {
	SetupEndpoints(app *fiber.App) error
}

// Command represents a CLI command provided by a plugin
type Command interface {
	// Name returns the command name (e.g., "models", "resources")
	Name() string

	// Description returns a brief description of what the command does
	Description() string

	// Run executes the command and returns a result
	Run(ctx *CommandContext) *CommandResult
}

// CommandContext provides context and dependencies for command execution
type CommandContext struct {
	// Config is the application configuration
	Config interface{}

	// Args are command-line arguments passed to the command
	Args []string

	// ProgressCallback is called to report progress
	ProgressCallback func(message string)
}

// CommandResult contains the result of command execution
type CommandResult struct {
	// Success indicates whether the command succeeded
	Success bool

	// Error contains the error if the command failed
	Error error

	// FilesCreated lists all files created by the command
	FilesCreated []string

	// FilesModified lists all files modified by the command
	FilesModified []string

	// Message contains additional information about the result
	Message string
}

// CommandProvider is an optional interface for plugins that provide CLI commands
type CommandProvider interface {
	// Commands returns all commands provided by this plugin
	Commands() []Command
}
