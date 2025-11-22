package requestid

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/pluginloader"
)

type RequestIDPlugin struct{}

func init() {
	pluginloader.RegisterGlobalPluginFactory("requestid", func() plugin.GlobalPlugin {
		return &RequestIDPlugin{}
	})
}

func (p *RequestIDPlugin) Name() string {
	return "requestid"
}

func (p *RequestIDPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (p *RequestIDPlugin) Handler() fiber.Handler {
	return requestid.New()
}
