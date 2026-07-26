package response

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/serializer"
)

var version = "dev"

// Precomputed: every response sets this header.
var poweredBy = "GoREST/" + version

func Initialize(v string) {
	version = v
	poweredBy = "GoREST/" + version
}

func SetCommonHeaders(c fiber.Ctx) {
	c.Set("X-Powered-By", poweredBy)
}

func SetContentTypeHeader(c fiber.Ctx, format string) {
	if format == "jsonld" {
		c.Set("Content-Type", "application/ld+json")
	} else {
		c.Set("Content-Type", "application/json")
	}
}

func SendFormatted(c fiber.Ctx, statusCode int, data interface{}) error {
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

	c.Set("X-Powered-By", poweredBy)
	c.Set("Content-Type", s.ContentType())
	return c.Status(statusCode).Send(formatted)
}

func ParseExpandQuery(c fiber.Ctx) []string {
	var expand []string
	for key, value := range c.Request().URI().QueryArgs().All() {
		keyStr := string(key)
		if keyStr == "expand[]" {
			expand = append(expand, string(value))
		}
	}
	return expand
}

func DetermineFormat(c fiber.Ctx) string {
	// Walked in place; splitting would allocate on a path every response takes.
	accept := c.Get("Accept", "")

	for rest := accept; rest != ""; {
		var part string
		part, rest, _ = strings.Cut(rest, ",")
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mediaType, _, _ := strings.Cut(part, ";")
		if mediaType == "" {
			continue
		}

		switch {
		case strings.Contains(mediaType, "application/json") && !strings.Contains(mediaType, "application/ld+json"):
			return "json"
		case strings.Contains(mediaType, "application/ld+json"):
			return "jsonld"
		}
	}

	return "jsonld"
}

func SendError(c fiber.Ctx, statusCode int, message string) error {
	c.Set("X-Powered-By", poweredBy)
	return c.Status(statusCode).JSON(fiber.Map{
		"error": message,
	})
}

// TODO refactor to more flexible Send method with status code
func SendSuccess(c fiber.Ctx, data interface{}) error {
	c.Set("X-Powered-By", poweredBy)
	return c.Status(fiber.StatusOK).JSON(data)
}

func SendCreated(c fiber.Ctx, data interface{}) error {
	c.Set("X-Powered-By", poweredBy)
	return c.Status(fiber.StatusCreated).JSON(data)
}

func SendJSON(c fiber.Ctx, statusCode int, data interface{}) error {
	c.Set("X-Powered-By", poweredBy)
	format := DetermineFormat(c)
	SetContentTypeHeader(c, format)
	SetCommonHeaders(c)
	return c.Status(statusCode).JSON(data)
}
