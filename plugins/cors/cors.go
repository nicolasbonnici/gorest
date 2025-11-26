package cors

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/nicolasbonnici/gorest/plugin"
)

type CORSPlugin struct {
	origins string
}

func NewPlugin() plugin.Plugin {
	return &CORSPlugin{
		origins: "*",
	}
}

func (p *CORSPlugin) Name() string {
	return "cors"
}

func (p *CORSPlugin) Initialize(config map[string]interface{}) error {
	if origins, ok := config["origins"].(string); ok {
		p.origins = origins
	}
	return nil
}

func (p *CORSPlugin) Handler() fiber.Handler {
	config := cors.Config{
		AllowOrigins: p.origins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		MaxAge:       86400,
	}

	if p.origins != "*" {
		config.AllowCredentials = true
	}

	return cors.New(config)
}
