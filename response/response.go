package response

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/serializer"
)

var version = "dev"

func Initialize(v string) {
	version = v
}

// SetCommonHeaders sets common response headers like X-Powered-By
func SetCommonHeaders(c *fiber.Ctx) {
	c.Set("X-Powered-By", "GoREST/"+version)
}

func SendFormatted(c *fiber.Ctx, statusCode int, data interface{}) error {
	format := DetermineFormat(c)
	s := serializer.GetSerializer(format)

	expand := ParseExpandQuery(c)
	var formatted []byte
	var err error

	if len(expand) > 0 {
		formatted, err = s.SerializeWithExpand(data, c.Path(), expand)
	} else {
		formatted, err = s.Serialize(data, c.Path())
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to serialize response"})
	}

	c.Set("X-Powered-By", "GoREST/"+version)
	c.Set("Content-Type", s.ContentType())
	return c.Status(statusCode).Send(formatted)
}

func ParseExpandQuery(c *fiber.Ctx) []string {
	var expand []string
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if keyStr == "expand[]" {
			expand = append(expand, string(value))
		}
	})
	return expand
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
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		mediaTypeParts := strings.Split(trimmed, ";")
		if len(mediaTypeParts) > 0 && mediaTypeParts[0] != "" {
			contentTypes = append(contentTypes, mediaTypeParts[0])
		}
	}

	return contentTypes
}

func SendError(c *fiber.Ctx, statusCode int, message string) error {
	c.Set("X-Powered-By", "GoREST/"+version)
	return c.Status(statusCode).JSON(fiber.Map{
		"error": message,
	})
}

func SendSuccess(c *fiber.Ctx, data interface{}) error {
	c.Set("X-Powered-By", "GoREST/"+version)
	return c.Status(fiber.StatusOK).JSON(data)
}

func SendCreated(c *fiber.Ctx, data interface{}) error {
	c.Set("X-Powered-By", "GoREST/"+version)
	return c.Status(fiber.StatusCreated).JSON(data)
}
