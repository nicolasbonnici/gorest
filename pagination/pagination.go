package pagination

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/response"
	"github.com/nicolasbonnici/gorest/serializer"
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

func buildPaginationURL(basePath string, params url.Values, limit, page, defaultLimit int) string {
	newParams := url.Values{}
	for k, v := range params {
		if k == "limit" || k == "page" {
			continue
		}

		if len(v) > 0 && v[0] != "" {
			newParams[k] = v
		}
	}

	if limit > 0 && limit != defaultLimit {
		newParams.Set("limit", strconv.Itoa(limit))
	}

	if page > 1 {
		newParams.Set("page", strconv.Itoa(page))
	}

	if len(newParams) > 0 {
		return basePath + "?" + newParams.Encode()
	}
	return basePath
}

func SendHydraCollectionWithExpanded(c *fiber.Ctx, expandedItems []interface{}, total *int, limit, page, defaultLimit int) error {
	basePath := c.Path()
	queryParams := c.Context().QueryArgs()
	parsedParams := make(url.Values)
	queryParams.VisitAll(func(key, value []byte) {
		parsedParams.Add(string(key), string(value))
	})

	currentURL := buildPaginationURL(basePath, parsedParams, limit, page, defaultLimit)

	view := &HydraView{
		ID:    currentURL,
		Type:  "hydra:PartialCollectionView",
		First: buildPaginationURL(basePath, parsedParams, limit, 1, defaultLimit),
	}

	if page > 1 {
		prevURL := buildPaginationURL(basePath, parsedParams, limit, page-1, defaultLimit)
		view.Previous = &prevURL
	}

	if total != nil {
		lastPage := (*total + limit - 1) / limit
		if page < lastPage {
			nextURL := buildPaginationURL(basePath, parsedParams, limit, page+1, defaultLimit)
			view.Next = &nextURL
		}

		lastURL := buildPaginationURL(basePath, parsedParams, limit, lastPage, defaultLimit)
		view.Last = &lastURL
	}

	format := response.DetermineFormat(c)
	s := serializer.GetSerializer(format)
	expand := response.ParseExpandQuery(c)

	formattedItems := make([]interface{}, len(expandedItems))
	for i, item := range expandedItems {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if format == "jsonld" {
				jsonBytes, _ := s.SerializeWithExpand(itemMap, basePath, expand)
				var enriched map[string]interface{}
				json.Unmarshal(jsonBytes, &enriched)
				delete(enriched, "@context")
				formattedItems[i] = enriched
			} else {
				formattedItems[i] = itemMap
			}
		} else {
			formattedItems[i] = item
		}
	}

	collection := HydraCollection{
		Context:    "http://www.w3.org/ns/hydra/context.jsonld",
		ID:         basePath,
		Type:       "hydra:Collection",
		TotalItems: total,
		Member:     formattedItems,
		View:       view,
	}

	if format == "jsonld" {
		c.Set("Content-Type", "application/ld+json")
	}

	response.SetCommonHeaders(c)
	return c.Status(fiber.StatusOK).JSON(collection)
}

func SendHydraCollection(c *fiber.Ctx, items interface{}, total *int, limit, page, defaultLimit int) error {
	basePath := c.Path()
	queryParams := c.Context().QueryArgs()
	parsedParams := make(url.Values)
	queryParams.VisitAll(func(key, value []byte) {
		parsedParams.Add(string(key), string(value))
	})

	currentURL := buildPaginationURL(basePath, parsedParams, limit, page, defaultLimit)

	view := &HydraView{
		ID:    currentURL,
		Type:  "hydra:PartialCollectionView",
		First: buildPaginationURL(basePath, parsedParams, limit, 1, defaultLimit),
	}

	if page > 1 {
		prevURL := buildPaginationURL(basePath, parsedParams, limit, page-1, defaultLimit)
		view.Previous = &prevURL
	}

	if total != nil {
		lastPage := (*total + limit - 1) / limit
		if page < lastPage {
			nextURL := buildPaginationURL(basePath, parsedParams, limit, page+1, defaultLimit)
			view.Next = &nextURL
		}

		lastURL := buildPaginationURL(basePath, parsedParams, limit, lastPage, defaultLimit)
		view.Last = &lastURL
	}

	expand := response.ParseExpandQuery(c)
	formattedItems := formatItems(items, basePath, response.DetermineFormat(c), expand)

	collection := HydraCollection{
		Context:    "http://www.w3.org/ns/hydra/context.jsonld",
		ID:         basePath,
		Type:       "hydra:Collection",
		TotalItems: total,
		Member:     formattedItems,
		View:       view,
	}

	format := c.Query("format", "json")
	if format == "jsonld" {
		c.Set("Content-Type", "application/ld+json")
	}

	response.SetCommonHeaders(c)
	return c.Status(fiber.StatusOK).JSON(collection)
}

func formatItems(items interface{}, path string, format string, expand []string) interface{} {
	val := reflect.ValueOf(items)
	if val.Kind() != reflect.Slice {
		return items
	}

	s := serializer.GetSerializer(format)
	formattedItems := make([]interface{}, val.Len())

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()

		if format == "jsonld" {
			jsonBytes, _ := s.SerializeWithExpand(item, path, expand)
			var itemMap map[string]interface{}
			json.Unmarshal(jsonBytes, &itemMap)

			delete(itemMap, "@context")
			formattedItems[i] = itemMap
		} else {
			jsonBytes, _ := json.Marshal(item)
			var itemMap map[string]interface{}
			json.Unmarshal(jsonBytes, &itemMap)
			formattedItems[i] = itemMap
		}
	}

	return formattedItems
}

func SendPaginatedError(c *fiber.Ctx, statusCode int, message string) error {
	response.SetCommonHeaders(c)
	return c.Status(statusCode).JSON(fiber.Map{
		"@context":          "http://www.w3.org/ns/hydra/context.jsonld",
		"@type":             "hydra:Error",
		"hydra:title":       fmt.Sprintf("Error %d", statusCode),
		"hydra:description": message,
	})
}
