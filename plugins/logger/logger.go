package logger

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/middleware"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/pluginloader"
)

type LoggerPlugin struct{}

func init() {
	pluginloader.RegisterGlobalPluginFactory("logger", func() plugin.GlobalPlugin {
		return &LoggerPlugin{}
	})
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
