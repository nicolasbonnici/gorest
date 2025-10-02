package gorest

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolasbonnici/gorest/internal"
)

type Config struct {
	DBUrl     string
	JWTSecret string
	Port      string
}

func Start(cfg Config) {
	db, err := pgxpool.New(context.Background(), cfg.DBUrl)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	tables := internal.LoadSchema(db)

	app := fiber.New()

	internal.SetupAuth(app, cfg.JWTSecret)

	internal.SetupAPI(app, db, tables, cfg.JWTSecret)

	internal.SetupOpenAPI(app, tables)

	log.Printf("🚀 REST API running at http://localhost:%s", cfg.Port)
	app.Listen(":" + cfg.Port)
}
