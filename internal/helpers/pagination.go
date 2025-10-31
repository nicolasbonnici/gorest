package helpers

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type HydraView struct {
	ID       string  `json:"@id"`
	Type     string  `json:"@type"`
	First    string  `json:"hydra:first"`
	Last     *string `json:"hydra:last,omitempty"`
	Previous *string `json:"hydra:previous,omitempty"`
	Next     *string `json:"hydra:next,omitempty"`
}

type HydraCollection struct {
	Context    string      `json:"@context"`
	ID         string      `json:"@id"`
	Type       string      `json:"@type"`
	TotalItems *int        `json:"hydra:totalItems,omitempty"`
	Member     interface{} `json:"hydra:member"`
	View       *HydraView  `json:"hydra:view"`
}

func ParseIntQuery(c *fiber.Ctx, key string, defaultValue, maxValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 0 {
		return defaultValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}

func buildPaginationURL(basePath string, params url.Values, limit, offset int) string {
	newParams := url.Values{}
	for k, v := range params {
		if k != "limit" && k != "offset" {
			newParams[k] = v
		}
	}
	newParams.Set("limit", strconv.Itoa(limit))
	newParams.Set("offset", strconv.Itoa(offset))

	if len(newParams) > 0 {
		return basePath + "?" + newParams.Encode()
	}
	return basePath
}

func SendHydraCollection(c *fiber.Ctx, items interface{}, total *int, limit, offset int) error {
	basePath := c.Path()
	queryParams := c.Context().QueryArgs()
	parsedParams := make(url.Values)
	queryParams.VisitAll(func(key, value []byte) {
		parsedParams.Add(string(key), string(value))
	})

	currentURL := buildPaginationURL(basePath, parsedParams, limit, offset)

	view := &HydraView{
		ID:    currentURL,
		Type:  "hydra:PartialCollectionView",
		First: buildPaginationURL(basePath, parsedParams, limit, 0),
	}

	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		prevURL := buildPaginationURL(basePath, parsedParams, limit, prevOffset)
		view.Previous = &prevURL
	}

	if total != nil {
		nextOffset := offset + limit
		if nextOffset < *total {
			nextURL := buildPaginationURL(basePath, parsedParams, limit, nextOffset)
			view.Next = &nextURL
		}

		lastOffset := (*total / limit) * limit
		if lastOffset == *total && *total > 0 {
			lastOffset = *total - limit
		}
		if lastOffset < 0 {
			lastOffset = 0
		}
		lastURL := buildPaginationURL(basePath, parsedParams, limit, lastOffset)
		view.Last = &lastURL
	}

	collection := HydraCollection{
		Context:    "http://www.w3.org/ns/hydra/context.jsonld",
		ID:         basePath,
		Type:       "hydra:Collection",
		TotalItems: total,
		Member:     items,
		View:       view,
	}

	format := c.Query("format", "json")
	if format == "jsonld" {
		c.Set("Content-Type", "application/ld+json")
	}

	return c.Status(fiber.StatusOK).JSON(collection)
}

func SendPaginatedError(c *fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"@context": "http://www.w3.org/ns/hydra/context.jsonld",
		"@type":    "hydra:Error",
		"hydra:title": fmt.Sprintf("Error %d", statusCode),
		"hydra:description": message,
	})
}
