package internal

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// pgToOpenAPIType converts PostgreSQL types to OpenAPI schema types
func pgToOpenAPIType(pgType string) (string, string) {
	typeMap := map[string]struct{ typ, format string }{
		"integer":                  {"integer", "int32"},
		"bigint":                   {"integer", "int64"},
		"smallint":                 {"integer", "int32"},
		"text":                     {"string", ""},
		"varchar":                  {"string", ""},
		"character varying":        {"string", ""},
		"boolean":                  {"boolean", ""},
		"timestamp without time zone": {"string", "date-time"},
		"timestamp with time zone": {"string", "date-time"},
		"timestamp":                {"string", "date-time"},
		"uuid":                     {"string", "uuid"},
		"numeric":                  {"number", "double"},
		"double precision":         {"number", "double"},
		"real":                     {"number", "float"},
		"json":                     {"object", ""},
		"jsonb":                    {"object", ""},
	}

	if mapping, ok := typeMap[pgType]; ok {
		return mapping.typ, mapping.format
	}
	return "string", ""
}

// buildSchemaProperties creates OpenAPI schema properties from table columns
func buildSchemaProperties(columns []Column) map[string]interface{} {
	properties := make(map[string]interface{})

	for _, col := range columns {
		typ, format := pgToOpenAPIType(col.Type)
		prop := map[string]interface{}{
			"type": typ,
		}

		if format != "" {
			prop["format"] = format
		}

		if !col.IsNullable {
			prop["nullable"] = false
		} else {
			prop["nullable"] = true
		}

		properties[col.Name] = prop
	}

	return properties
}

// getRequiredFields returns list of non-nullable fields (excluding auto-generated fields)
func getRequiredFields(columns []Column) []string {
	var required []string

	for _, col := range columns {
		// Skip auto-generated fields
		if col.Name == "id" || col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		if !col.IsNullable {
			required = append(required, col.Name)
		}
	}

	return required
}

func SetupOpenAPI(app *fiber.App, tables map[string]TableSchema) {
	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		paths := map[string]interface{}{}
		components := map[string]interface{}{
			"schemas": make(map[string]interface{}),
		}

		// Build component schemas for each table
		for _, t := range tables {
			singularName := SingularizeExported(t.TableName)
			schemaName := strings.ToUpper(singularName[:1]) + singularName[1:]

			properties := buildSchemaProperties(t.Columns)
			required := getRequiredFields(t.Columns)

			schema := map[string]interface{}{
				"type":       "object",
				"properties": properties,
			}

			if len(required) > 0 {
				schema["required"] = required
			}

			components["schemas"].(map[string]interface{})[schemaName] = schema
		}

		// Build paths with proper request/response schemas
		for _, t := range tables {
			singularName := SingularizeExported(t.TableName)
			schemaName := strings.ToUpper(singularName[:1]) + singularName[1:]
			base := "/" + t.TableName

			// List endpoint (GET /resource)
			paths[base] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List " + t.TableName,
					"description": "Retrieve a list of " + t.TableName,
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
				// Create endpoint (POST /resource)
				"post": map[string]interface{}{
					"summary":     "Create " + singularName,
					"description": "Create a new " + singularName,
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

			// Item-specific endpoints (GET/PUT/DELETE /resource/{id})
			paths[base+"/{id}"] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get " + singularName + " by ID",
					"description": "Retrieve a single " + singularName + " by ID",
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
					"summary":     "Update " + singularName + " by ID",
					"description": "Update an existing " + singularName,
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
					"summary":     "Delete " + singularName + " by ID",
					"description": "Delete an existing " + singularName,
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

			// Relationship endpoints
			for _, r := range t.Relations {
				subPath := "/" + r.ParentTable + "/{id}/" + r.ChildTable
				childSingular := SingularizeExported(r.ChildTable)
				childSchema := strings.ToUpper(childSingular[:1]) + childSingular[1:]

				paths[subPath] = map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get " + r.ChildTable + " by parent " + r.ParentTable,
						"description": "Retrieve " + r.ChildTable + " related to a " + singularName,
						"tags":        []string{schemaName},
						"parameters": []map[string]interface{}{
							{
								"name":        "id",
								"in":          "path",
								"required":    true,
								"description": "Parent resource ID",
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
												"$ref": "#/components/schemas/" + childSchema,
											},
										},
									},
								},
							},
						},
					},
				}
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
