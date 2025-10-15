package internal

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/internal/formatter"
)

func SendFormatted(c *fiber.Ctx, statusCode int, data interface{}) error {
	format := DetermineFormat(c)
	f := formatter.GetFormatter(format)

	formatted, err := f.Format(data, c.Path())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to format response"})
	}

	c.Set("Content-Type", f.ContentType())
	return c.Status(statusCode).Send(formatted)
}

func DetermineFormat(c *fiber.Ctx) string {
	accept := c.Get("Accept", "")
	contentTypes := parseAcceptHeader(accept)

	for _, ct := range contentTypes {
		switch {
		case strings.Contains(ct, "application/json") && !strings.Contains(ct, "application/ld+json"):
			return "json"
		case strings.Contains(ct, "application/ld+json"):
			return "jsonld"
		}
	}

	return "jsonld"
}

func parseAcceptHeader(accept string) []string {
	if accept == "" {
		return []string{}
	}

	parts := strings.Split(accept, ",")
	contentTypes := make([]string, 0, len(parts))

	for _, part := range parts {
		ct := strings.Split(strings.TrimSpace(part), ";")[0]
		contentTypes = append(contentTypes, ct)
	}

	return contentTypes
}
