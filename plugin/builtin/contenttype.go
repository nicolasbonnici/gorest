package builtin

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

// ContentTypePlugin validates Content-Type header for mutating requests
type ContentTypePlugin struct{}

func NewContentTypePlugin() plugin.GlobalPlugin {
	return &ContentTypePlugin{}
}

func (p *ContentTypePlugin) Name() string {
	return "contenttype"
}

func (p *ContentTypePlugin) Initialize(config map[string]interface{}) error {
	// No configuration needed
	return nil
}

func (p *ContentTypePlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
					"error": "Content-Type must be application/json",
				})
			}
		}
		return c.Next()
	}
}
