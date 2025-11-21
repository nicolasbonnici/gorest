package builtin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/middleware"
	"github.com/nicolasbonnici/gorest/plugin"
)

// LoggerPlugin provides HTTP request/response logging
type LoggerPlugin struct{}

func NewLoggerPlugin() plugin.GlobalPlugin {
	return &LoggerPlugin{}
}

func (p *LoggerPlugin) Name() string {
	return "logger"
}

func (p *LoggerPlugin) Initialize(config map[string]interface{}) error {
	// No configuration needed
	return nil
}

func (p *LoggerPlugin) Handler() fiber.Handler {
	return middleware.HTTPLogger()
}
