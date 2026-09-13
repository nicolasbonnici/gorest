package response

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/serializer"
)

var version = "dev"

// Precomputed: every response sets this header.
var poweredBy = "GoREST/" + version

// Initialize records the framework version used in the X-Powered-By header.
func Initialize(v string) {
	version = v
	poweredBy = "GoREST/" + version
}

// DisablePoweredBy stops advertising the framework and its exact version on
// every response. A version string is the first thing an attacker looks up
// against a CVE list, and it buys a client nothing, so gorest.Start turns the
// header off outside development.
func DisablePoweredBy() {
	poweredBy = ""
}

func setPoweredBy(c fiber.Ctx) {
	if poweredBy != "" {
		c.Set("X-Powered-By", poweredBy)
	}
}

func SetCommonHeaders(c fiber.Ctx) {
	setPoweredBy(c)
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

	setPoweredBy(c)
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
	setPoweredBy(c)
	return c.Status(statusCode).JSON(fiber.Map{
		"error": message,
	})
}

// TODO refactor to more flexible Send method with status code
func SendSuccess(c fiber.Ctx, data interface{}) error {
	setPoweredBy(c)
	return c.Status(fiber.StatusOK).JSON(data)
}

func SendCreated(c fiber.Ctx, data interface{}) error {
	setPoweredBy(c)
	return c.Status(fiber.StatusCreated).JSON(data)
}

func SendJSON(c fiber.Ctx, statusCode int, data interface{}) error {
	setPoweredBy(c)
	format := DetermineFormat(c)
	SetContentTypeHeader(c, format)
	SetCommonHeaders(c)
	return c.Status(statusCode).JSON(data)
}
