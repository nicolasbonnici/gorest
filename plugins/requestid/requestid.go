package requestid

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/nicolasbonnici/gorest/plugin"
)

type RequestIDPlugin struct{}

func NewPlugin() plugin.GlobalPlugin {
	return &RequestIDPlugin{}
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
