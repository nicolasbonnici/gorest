package openapi

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/codegen"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

// OpenAPIPlugin provides runtime OpenAPI schema serving
type OpenAPIPlugin struct {
	db                 database.Database
	paginationLimit    int
	paginationMaxLimit int
	tables             map[string]codegen.TableSchema
}

func NewPlugin() plugin.Plugin {
	return &OpenAPIPlugin{}
}

func (p *OpenAPIPlugin) Name() string {
	return "openapi"
}

func (p *OpenAPIPlugin) Initialize(cfg map[string]interface{}) error {
	if db, ok := cfg["database"].(database.Database); ok {
		p.db = db

		// Load schema from database
		schemaSlice, err := db.Introspector().LoadSchema(context.Background())
		if err != nil {
			return err
		}

		// Convert to table schemas
		p.tables = make(map[string]codegen.TableSchema)
		for _, t := range schemaSlice {
			p.tables[t.TableName] = codegen.TableSchema{
				TableName: t.TableName,
				Columns:   convertColumns(t.Columns),
				Relations: convertRelations(t.Relations),
			}
		}
	}

	if limit, ok := cfg["pagination_limit"].(int); ok {
		p.paginationLimit = limit
	}
	if maxLimit, ok := cfg["pagination_max_limit"].(int); ok {
		p.paginationMaxLimit = maxLimit
	}

	return nil
}

// Handler returns a no-op middleware
func (p *OpenAPIPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// SetupEndpoints implements the EndpointSetup interface
func (p *OpenAPIPlugin) SetupEndpoints(app *fiber.App) error {
	// Setup OpenAPI UI endpoint
	app.Get("/openapi", func(c *fiber.Ctx) error {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>GoREST API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <script id="api-reference" data-url="/openapi.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})

	// Setup dynamic OpenAPI JSON endpoint
	codegen.SetupOpenAPI(app, p.tables, p.paginationLimit, p.paginationMaxLimit)
	return nil
}

func convertColumns(dbCols []database.Column) []codegen.Column {
	cols := make([]codegen.Column, len(dbCols))
	for i, c := range dbCols {
		cols[i] = codegen.Column{
			Name:       c.Name,
			Type:       c.Type,
			IsNullable: c.IsNullable,
		}
	}
	return cols
}

func convertRelations(dbRels []database.Relation) []codegen.Relation {
	rels := make([]codegen.Relation, len(dbRels))
	for i, r := range dbRels {
		rels[i] = codegen.Relation{
			ChildTable:   r.ChildTable,
			ChildColumn:  r.ChildColumn,
			ParentTable:  r.ParentTable,
			ParentColumn: r.ParentColumn,
		}
	}
	return rels
}
