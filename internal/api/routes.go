package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api/resources"
)

func RegisterGeneratedRoutes(app *fiber.App, db *pgxpool.Pool, tables map[string]internal.TableSchema) {
	for tableName := range tables {
		switch tableName {
		case "users":
			resources.RegisterUserRoutes(app, db)
		case "todo":
			resources.RegisterTodoRoutes(app, db)
		}
	}
}
