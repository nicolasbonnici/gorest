package internal

import (
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

func SetupOpenAPI(app *fiber.App, tables map[string]TableSchema) {
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
							"description": "Maximum number of items to return",
							"schema":      map[string]string{"type": "integer"},
						},
						{
							"name":        "offset",
							"in":          "query",
							"description": "Number of items to skip",
							"schema":      map[string]string{"type": "integer"},
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
							"description": "Successful response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]string{
											"$ref": "#/components/schemas/" + schemaName,
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

		return c.JSON(map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]interface{}{
				"title":       "GoREST API",
				"version":     "1.0.0",
				"description": "Auto-generated REST API with full CRUD operations",
			},
			"paths":      paths,
			"components": components,
		})
	})
}
