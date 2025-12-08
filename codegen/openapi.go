package codegen

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

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
											"@context": map[string]string{"type": "string"},
											"@id":      map[string]string{"type": "string"},
											"@type":    map[string]string{"type": "string", "example": "hydra:Collection"},
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

		paths["/login"] = map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "User login",
				"description": "Authenticate a user and receive a JWT token",
				"tags":        []string{"Authentication"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"required": []string{"email", "password"},
								"properties": map[string]interface{}{
									"email": map[string]interface{}{
										"type":        "string",
										"format":      "email",
										"description": "User email address",
									},
									"password": map[string]interface{}{
										"type":        "string",
										"format":      "password",
										"description": "User password",
									},
								},
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful authentication",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"token": map[string]interface{}{
											"type":        "string",
											"description": "JWT authentication token",
										},
										"user": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"id": map[string]interface{}{
													"type": "string",
												},
												"email": map[string]interface{}{
													"type": "string",
												},
												"firstname": map[string]interface{}{
													"type": "string",
												},
												"lastname": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
							},
						},
					},
					"400": map[string]interface{}{
						"description": "Bad request - missing email or password",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"error": map[string]interface{}{
											"type": "string",
										},
									},
								},
							},
						},
					},
					"401": map[string]interface{}{
						"description": "Unauthorized - invalid credentials",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"error": map[string]interface{}{
											"type": "string",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		paths["/health"] = map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Health check",
				"description": "Check API and database health status",
				"tags":        []string{"System"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Service is healthy",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"status": map[string]interface{}{
											"type":        "string",
											"enum":        []string{"healthy"},
											"description": "Overall health status",
										},
										"database": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"status": map[string]interface{}{
													"type": "string",
													"enum": []string{"up"},
												},
											},
										},
									},
								},
							},
						},
					},
					"503": map[string]interface{}{
						"description": "Service is unhealthy",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"status": map[string]interface{}{
											"type": "string",
											"enum": []string{"unhealthy"},
										},
										"database": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"status": map[string]interface{}{
													"type": "string",
													"enum": []string{"down"},
												},
												"error": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		components["securitySchemes"] = map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "JWT token from /login endpoint",
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
