package codegen

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/plugins/codegen/generator"
)

// CodegenPlugin provides code generation commands for models, resources, DTOs, and OpenAPI schemas
type CodegenPlugin struct {
	db     database.Database
	config *config.Config
}

func NewPlugin() plugin.Plugin {
	return &CodegenPlugin{}
}

func (p *CodegenPlugin) Name() string {
	return "codegen"
}

func (p *CodegenPlugin) Initialize(cfg map[string]interface{}) error {
	if db, ok := cfg["database"].(database.Database); ok {
		p.db = db
	}
	if appConfig, ok := cfg["config"].(*config.Config); ok {
		p.config = appConfig
	}
	return nil
}

// Handler returns a no-op middleware since codegen is a build-time plugin
func (p *CodegenPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// Commands implements the CommandProvider interface
func (p *CodegenPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		&ModelGenCommand{plugin: p},
		&ResourceGenCommand{plugin: p},
		&OpenAPIGenCommand{plugin: p},
		&AllCommand{plugin: p},
	}
}

// ModelGenCommand generates model structs from database schema
type ModelGenCommand struct {
	plugin *CodegenPlugin
}

func (c *ModelGenCommand) Name() string {
	return "models"
}

func (c *ModelGenCommand) Description() string {
	return "Generate model structs from database schema"
}

func (c *ModelGenCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Loading database schema...")
	}

	tables := generator.LoadSchema(c.plugin.db)

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Generating model structs...")
	}

	generator.GenerateStructs(tables)

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Models generated successfully")
	}

	return &plugin.CommandResult{
		Success: true,
		Message: "Model generation completed successfully",
	}
}

// ResourceGenCommand generates REST API resources from models
type ResourceGenCommand struct {
	plugin *CodegenPlugin
}

func (c *ResourceGenCommand) Name() string {
	return "resources"
}

func (c *ResourceGenCommand) Description() string {
	return "Generate REST API resources and DTOs from models"
}

func (c *ResourceGenCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Generating API resources...")
	}

	// Use default auth configuration
	authCfg := generator.DefaultAuthConfig()

	generator.GenerateAPI(authCfg)

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Resources generated successfully")
	}

	return &plugin.CommandResult{
		Success: true,
		Message: "Resource generation completed successfully",
	}
}

// OpenAPIGenCommand generates OpenAPI schema file
type OpenAPIGenCommand struct {
	plugin *CodegenPlugin
}

func (c *OpenAPIGenCommand) Name() string {
	return "openapi"
}

func (c *OpenAPIGenCommand) Description() string {
	return "Generate OpenAPI schema file"
}

func (c *OpenAPIGenCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("Generating OpenAPI schema...")
	}

	tables := generator.LoadSchema(c.plugin.db)
	generator.GenerateOpenAPI(tables)

	if ctx.ProgressCallback != nil {
		ctx.ProgressCallback("OpenAPI schema generated successfully")
	}

	return &plugin.CommandResult{
		Success: true,
		Message: "OpenAPI generation completed successfully",
	}
}

// AllCommand runs all generation commands in sequence
type AllCommand struct {
	plugin *CodegenPlugin
}

func (c *AllCommand) Name() string {
	return "all"
}

func (c *AllCommand) Description() string {
	return "Run all code generation steps (models, resources, openapi)"
}

func (c *AllCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	commands := []plugin.Command{
		&ModelGenCommand{plugin: c.plugin},
		&ResourceGenCommand{plugin: c.plugin},
		&OpenAPIGenCommand{plugin: c.plugin},
	}

	for _, cmd := range commands {
		if ctx.ProgressCallback != nil {
			ctx.ProgressCallback("Running: " + cmd.Name())
		}

		result := cmd.Run(ctx)
		if !result.Success {
			return result
		}
	}

	return &plugin.CommandResult{
		Success: true,
		Message: "All code generation completed successfully",
	}
}
