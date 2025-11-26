package logger

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/middleware"
	"github.com/nicolasbonnici/gorest/plugin"
)

type LoggerPlugin struct{}

func NewPlugin() plugin.Plugin {
	return &LoggerPlugin{}
}

func (p *LoggerPlugin) Name() string {
	return "logger"
}

func (p *LoggerPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (p *LoggerPlugin) Handler() fiber.Handler {
	return middleware.HTTPLogger()
}
