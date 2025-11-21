package builtin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/nicolasbonnici/gorest/plugin"
)

// RequestIDPlugin adds unique request ID tracking to each request
type RequestIDPlugin struct{}

func NewRequestIDPlugin() plugin.GlobalPlugin {
	return &RequestIDPlugin{}
}

func (p *RequestIDPlugin) Name() string {
	return "requestid"
}

func (p *RequestIDPlugin) Initialize(config map[string]interface{}) error {
	// No configuration needed
	return nil
}

func (p *RequestIDPlugin) Handler() fiber.Handler {
	return requestid.New()
}
