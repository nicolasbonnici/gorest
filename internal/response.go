package internal

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/internal/formatter"
)

// SendFormatted sends a response in the appropriate format based on Accept header
// Defaults to JSON-LD if no Accept header or unrecognized format
func SendFormatted(c *fiber.Ctx, statusCode int, data interface{}) error {
	format := DetermineFormat(c)
	f := formatter.GetFormatter(format)

	formatted, err := f.Format(data)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to format response"})
	}

	c.Set("Content-Type", f.ContentType())
	return c.Status(statusCode).Send(formatted)
}

// DetermineFormat determines the response format based on Accept header
// Returns "json" or "jsonld" (default)
func DetermineFormat(c *fiber.Ctx) string {
	accept := c.Get("Accept", "")

	// Parse Accept header for content types
	contentTypes := parseAcceptHeader(accept)

	for _, ct := range contentTypes {
		switch {
		case strings.Contains(ct, "application/json") && !strings.Contains(ct, "application/ld+json"):
			return "json"
		case strings.Contains(ct, "application/ld+json"):
			return "jsonld"
		}
	}

	// Default to JSON-LD
	return "jsonld"
}

// parseAcceptHeader parses the Accept header into content types
func parseAcceptHeader(accept string) []string {
	if accept == "" {
		return []string{}
	}

	parts := strings.Split(accept, ",")
	contentTypes := make([]string, 0, len(parts))

	for _, part := range parts {
		// Remove quality values (e.g., ";q=0.9")
		ct := strings.Split(strings.TrimSpace(part), ";")[0]
		contentTypes = append(contentTypes, ct)
	}

	return contentTypes
}
