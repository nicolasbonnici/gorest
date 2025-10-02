package internal

import (
	"github.com/gofiber/fiber/v2"
)

func SetupOpenAPI(app *fiber.App, tables map[string]TableSchema) {
	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		paths := map[string]interface{}{}

		for _, t := range tables {
			base := "/" + t.TableName
			paths[base] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List " + t.TableName,
					"parameters": []map[string]string{
						{"name": "limit", "in": "query"},
						{"name": "offset", "in": "query"},
						{"name": "expand", "in": "query"},
					},
				},
				"post": map[string]string{"summary": "Create " + t.TableName},
			}
			paths[base+"/{id}"] = map[string]string{
				"get":    "Get " + t.TableName + " by ID",
				"put":    "Update " + t.TableName + " by ID",
				"delete": "Delete " + t.TableName + " by ID",
			}

			for _, r := range t.Relations {
				subPath := "/" + r.ParentTable + "/{id}/" + r.ChildTable
				paths[subPath] = map[string]string{"get": "Get " + r.ChildTable + " by parent " + r.ParentTable}
			}
		}

		return c.JSON(map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]string{
				"title":   "Generic REST API",
				"version": "1.0.0",
			},
			"paths": paths,
		})
	})
}
