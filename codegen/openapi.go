package codegen

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// discoverNonResourceRoutes inspects the Fiber app to find routes not handled by resource DTOs
func discoverNonResourceRoutes(app *fiber.App, resourcePaths map[string]bool) map[string]map[string]interface{} {
	routes := app.GetRoutes(true) // Filter middleware-only routes
	discovered := make(map[string]map[string]interface{})

	for _, route := range routes {
		path := route.Path
		method := strings.ToUpper(route.Method)

		// Skip routes we don't want to document
		if shouldSkipRoute(path, resourcePaths) {
			continue
		}

		// Group by path
		if discovered[path] == nil {
			discovered[path] = make(map[string]interface{})
		}

		// Add method to this path
		discovered[path][strings.ToLower(method)] = generateRouteSpec(path, method)
	}

	return discovered
}

// shouldSkipRoute determines if a route should be excluded from OpenAPI docs
func shouldSkipRoute(path string, resourcePaths map[string]bool) bool {
	// Skip OpenAPI routes
	if path == "/openapi" || path == "/openapi.json" {
		return true
	}

	// Skip resource routes (already handled)
	if resourcePaths[path] {
		return true
	}

	// Skip empty paths
	if path == "" || path == "/" {
		return true
	}

	return false
}

// generateRouteSpec creates a basic OpenAPI spec for a discovered route
func generateRouteSpec(path, method string) map[string]interface{} {
	// Determine tag from path (e.g., /auth/login -> Authentication)
	tag := determineTag(path)

	// Generate summary and description
	summary := generateSummary(path, method)
	description := generateDescription(path, method)

	spec := map[string]interface{}{
		"summary":     summary,
		"description": description,
		"tags":        []string{tag},
	}

	// Add parameters for path params
	if strings.Contains(path, ":") {
		spec["parameters"] = extractPathParameters(path)
	}

	// Add request body for POST/PUT/PATCH
	if method == "POST" || method == "PUT" || method == "PATCH" {
		spec["requestBody"] = generateRequestBody(path)
	}

	// Add responses
	spec["responses"] = generateResponses(method)

	return spec
}

// determineTag extracts a tag name from the route path
func determineTag(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "General"
	}

	// Use the first path segment as tag
	segment := parts[0]

	// Capitalize and clean up
	switch segment {
	case "auth":
		return "Authentication"
	case "health":
		return "System"
	default:
		return strings.ToUpper(segment[:1]) + segment[1:]
	}
}

// generateSummary creates a human-readable summary for a route
func generateSummary(path, method string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	action := ""

	switch method {
	case "GET":
		action = "Get"
	case "POST":
		action = "Create or execute"
	case "PUT":
		action = "Update"
	case "PATCH":
		action = "Partially update"
	case "DELETE":
		action = "Delete"
	default:
		action = method
	}

	// Create readable path name
	pathName := strings.Join(parts, " ")
	pathName = strings.ReplaceAll(pathName, ":", "")

	return fmt.Sprintf("%s %s", action, pathName)
}

// generateDescription creates a description for a route
func generateDescription(path, method string) string {
	return fmt.Sprintf("%s %s", method, path)
}

// extractPathParameters extracts parameters from a Fiber route path
func extractPathParameters(path string) []map[string]interface{} {
	var params []map[string]interface{}

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			params = append(params, map[string]interface{}{
				"name":        paramName,
				"in":          "path",
				"required":    true,
				"description": fmt.Sprintf("Path parameter: %s", paramName),
				"schema":      map[string]string{"type": "string"},
			})
		}
	}

	return params
}

// generateRequestBody creates a generic request body spec
func generateRequestBody(path string) map[string]interface{} {
	return map[string]interface{}{
		"required": true,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}
}

// generateResponses creates standard responses for a method
func generateResponses(method string) map[string]interface{} {
	responses := map[string]interface{}{}

	switch method {
	case "GET":
		responses["200"] = map[string]interface{}{
			"description": "Successful response",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
		}
	case "POST":
		responses["201"] = map[string]interface{}{
			"description": "Successfully created",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
		}
		responses["400"] = map[string]interface{}{
			"description": "Bad request",
		}
	case "PUT", "PATCH":
		responses["200"] = map[string]interface{}{
			"description": "Successfully updated",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
		}
		responses["404"] = map[string]interface{}{
			"description": "Not found",
		}
	case "DELETE":
		responses["204"] = map[string]interface{}{
			"description": "Successfully deleted",
		}
		responses["404"] = map[string]interface{}{
			"description": "Not found",
		}
	default:
		responses["200"] = map[string]interface{}{
			"description": "Successful response",
		}
	}

	return responses
}

func buildSchemaPropertiesFromDTO(fields []StructField) map[string]interface{} {
	properties := make(map[string]interface{})

	for _, field := range fields {
		typ, format := GoTypeToOpenAPIType(field.Type)
		prop := map[string]interface{}{
			"type": typ,
		}

		if format != "" {
			prop["format"] = format
		}

		prop["nullable"] = field.IsPointer

		jsonName := field.JSONTag
		if jsonName == "" {
			jsonName = strings.ToLower(field.Name)
		}

		properties[jsonName] = prop
	}

	return properties
}

func getRequiredFieldsFromDTO(fields []StructField) []string {
	var required []string

	for _, field := range fields {
		jsonName := field.JSONTag
		if jsonName == "" {
			jsonName = strings.ToLower(field.Name)
		}

		if jsonName == "id" || jsonName == "created_at" || jsonName == "updated_at" {
			continue
		}

		if !field.IsPointer {
			required = append(required, jsonName)
		}
	}

	return required
}

func SetupOpenAPI(app *fiber.App, tables map[string]TableSchema, paginationLimit, paginationMaxLimit int) {
	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		resourceDTOs := LoadResourceDTOs()

		paths := map[string]interface{}{}
		components := map[string]interface{}{
			"schemas": make(map[string]interface{}),
		}

		// Track resource paths so we can filter them out from discovery
		resourcePaths := make(map[string]bool)

		for _, resource := range resourceDTOs {
			mainDTO := resource.GetMainDTO()
			if mainDTO == nil {
				continue
			}

			schemaName := strings.ToUpper(resource.Name[:1]) + resource.Name[1:]
			properties := buildSchemaPropertiesFromDTO(mainDTO.Fields)
			required := getRequiredFieldsFromDTO(mainDTO.Fields)

			schema := map[string]interface{}{
				"type":       "object",
				"properties": properties,
			}

			if len(required) > 0 {
				schema["required"] = required
			}

			components["schemas"].(map[string]interface{})[schemaName] = schema
		}

		for _, resource := range resourceDTOs {
			schemaName := strings.ToUpper(resource.Name[:1]) + resource.Name[1:]
			base := "/" + resource.PluralName

			// Mark resource paths
			resourcePaths[base] = true
			resourcePaths[base+"/:id"] = true

			paths[base] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List " + resource.PluralName,
					"description": "Retrieve a list of " + resource.PluralName,
					"tags":        []string{schemaName},
					"parameters": []map[string]interface{}{
						{
							"name":        "limit",
							"in":          "query",
							"description": fmt.Sprintf("Maximum number of items to return (default: %d, max: %d)", paginationLimit, paginationMaxLimit),
							"schema":      map[string]interface{}{"type": "integer", "default": paginationLimit, "maximum": paginationMaxLimit},
						},
						{
							"name":        "offset",
							"in":          "query",
							"description": "Number of items to skip (default: 0)",
							"schema":      map[string]interface{}{"type": "integer", "default": 0, "minimum": 0},
						},
						{
							"name":        "count",
							"in":          "query",
							"description": "Include total count in response (adds hydra:totalItems field)",
							"schema":      map[string]interface{}{"type": "boolean", "default": false},
						},
						{
							"name":        "expand",
							"in":          "query",
							"description": "Comma-separated list of relations to expand",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Hydra paginated collection",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"@context":         map[string]string{"type": "string"},
											"@id":              map[string]string{"type": "string"},
											"@type":            map[string]string{"type": "string", "example": "hydra:Collection"},
											"hydra:totalItems": map[string]interface{}{"type": "integer", "description": "Total count (only present if count=true)"},
											"hydra:member": map[string]interface{}{
												"type": "array",
												"items": map[string]string{
													"$ref": "#/components/schemas/" + schemaName,
												},
											},
											"hydra:view": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"@id":            map[string]string{"type": "string"},
													"@type":          map[string]string{"type": "string"},
													"hydra:first":    map[string]string{"type": "string"},
													"hydra:last":     map[string]string{"type": "string"},
													"hydra:previous": map[string]string{"type": "string"},
													"hydra:next":     map[string]string{"type": "string"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create " + resource.Name,
					"description": "Create a new " + resource.Name,
					"tags":        []string{schemaName},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/" + schemaName,
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Successfully created",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{
										"$ref": "#/components/schemas/" + schemaName,
									},
								},
							},
						},
					},
				},
			}

			paths[base+"/{id}"] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get " + resource.Name + " by ID",
					"description": "Retrieve a single " + resource.Name + " by ID",
					"tags":        []string{schemaName},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"required":    true,
							"description": "Resource ID",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{
										"$ref": "#/components/schemas/" + schemaName,
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Resource not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update " + resource.Name + " by ID",
					"description": "Update an existing " + resource.Name,
					"tags":        []string{schemaName},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"required":    true,
							"description": "Resource ID",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/" + schemaName,
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successfully updated",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{
										"$ref": "#/components/schemas/" + schemaName,
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Resource not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete " + resource.Name + " by ID",
					"description": "Delete an existing " + resource.Name,
					"tags":        []string{schemaName},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"required":    true,
							"description": "Resource ID",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "Successfully deleted",
						},
						"404": map[string]interface{}{
							"description": "Resource not found",
						},
					},
				},
			}
		}

		// Discover and add non-resource routes (e.g., /auth/login, /health, etc.)
		discoveredRoutes := discoverNonResourceRoutes(app, resourcePaths)
		for path, methods := range discoveredRoutes {
			paths[path] = methods
		}

		components["securitySchemes"] = map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "JWT authentication token",
			},
		}

		return c.JSON(map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]interface{}{
				"title":       "GoREST API",
				"version":     "1.0.0",
				"description": "Auto-generated REST API with full CRUD operations",
			},
			"servers": []map[string]string{
				{"url": "http://localhost:3000", "description": "Development server"},
			},
			"paths":      paths,
			"components": components,
			"security": []map[string]interface{}{
				{"bearerAuth": []string{}},
			},
		})
	})
}
