package gorest

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/nicolasbonnici/gorest/auth"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/generator"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/middleware"
)

// Config holds the path to configuration file and optional route registration function
type Config struct {
	// ConfigPath is the directory containing gorest.yaml (default: ".")
	ConfigPath string
	// RegisterRoutes is an optional callback to register generated routes
	RegisterRoutes func(app *fiber.App, db database.Database, jwtSecret string, paginationLimit, paginationMaxLimit int)
}

func Start(cfg Config) {
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "."
	}

	appConfig, err := config.Load(cfg.ConfigPath)
	if err != nil {
		logger.Log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := appConfig.Validate(); err != nil {
		logger.Log.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	checkGeneratedCode(appConfig)

	db, err := database.Open("", appConfig.Database.URL)
	if err != nil {
		logger.Log.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	schemaSlice, err := db.Introspector().LoadSchema(context.Background())
	if err != nil {
		logger.Log.Error("Failed to load schema", "error", err)
		os.Exit(1)
	}

	tables := make(map[string]generator.TableSchema)
	for _, t := range schemaSlice {
		tables[t.TableName] = generator.TableSchema{
			TableName: t.TableName,
			Columns:   convertColumns(t.Columns),
			Relations: convertRelations(t.Relations),
		}
	}

	app := fiber.New(fiber.Config{
		BodyLimit:    4 * 1024 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(requestid.New())

	if appConfig.RateLimit.Enabled {
		app.Use(limiter.New(limiter.Config{
			Max:        appConfig.RateLimit.RequestsPerSecond,
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(429).JSON(fiber.Map{
					"error": "Rate limit exceeded. Please try again later.",
				})
			},
		}))
	}

	corsOrigins := appConfig.GetCORSOrigins()
	corsOriginsStr := strings.Join(corsOrigins, ",")
	if corsOriginsStr == "*" {
		logger.Log.Warn("CORS_ORIGINS is set to '*' (allow all). Not recommended for production!")
	}

	corsConfig := cors.Config{
		AllowOrigins: corsOriginsStr,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}

	if corsOriginsStr != "*" {
		corsConfig.AllowCredentials = true
	}

	app.Use(cors.New(corsConfig))

	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	})

	app.Use(func(c *fiber.Ctx) error {
		method := c.Method()
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.Get("Content-Type")
			if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
				return c.Status(415).JSON(fiber.Map{
					"error": "Content-Type must be application/json",
				})
			}
		}
		return c.Next()
	})

	app.Use(middleware.HTTPLogger())

	SetupOpenAPIUI(app)
	SetupHealthCheck(app, db, logger.Log)
	auth.SetupAuth(app, db, appConfig.Auth.JWT.Secret, appConfig.Auth.JWT.TTL)

	if cfg.RegisterRoutes != nil {
		cfg.RegisterRoutes(app, db, appConfig.Auth.JWT.Secret, appConfig.Pagination.DefaultLimit, appConfig.Pagination.MaxLimit)
	} else {
		logger.Log.Warn("No routes registered. Set Config.RegisterRoutes to register your API endpoints.")
	}

	generator.SetupOpenAPI(app, tables, appConfig.Pagination.DefaultLimit, appConfig.Pagination.MaxLimit)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := fmt.Sprintf("%d", appConfig.Server.Port)
		logger.Log.Info("REST API running", "port", port, "url", "http://localhost:"+port)
		logger.Log.Info("Health check available", "url", "http://localhost:"+port+"/health")
		logger.Log.Info("Environment", "env", appConfig.Server.Environment)
		if err := app.Listen(":" + port); err != nil {
			logger.Log.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Log.Info("Shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Log.Warn("Server forced to shutdown", "error", err)
	}

	db.Close()
	logger.Log.Info("Server shutdown complete")
}

func checkGeneratedCode(cfg *config.Config) {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		logger.Log.Error("Failed to find project root", "error", err)
		os.Exit(1)
	}

	modelsDir := filepath.Join(projectRoot, cfg.Generate.Output.Models)
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		logger.Log.Error("Models not found. Run model generation first to generate models from database schema",
			"expected_path", modelsDir)
		os.Exit(1)
	}

	resourcesDir := filepath.Join(projectRoot, cfg.Generate.Output.Resources)
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		logger.Log.Error("Resources not found. Run resource generation first to generate API resources",
			"expected_path", resourcesDir)
		os.Exit(1)
	}

	openapiDir := filepath.Join(projectRoot, cfg.Generate.Output.OpenAPI)
	if _, err := os.Stat(openapiDir); os.IsNotExist(err) {
		logger.Log.Error("OpenAPI schema not found. Run OpenAPI generation first to generate OpenAPI schema",
			"expected_path", openapiDir)
		os.Exit(1)
	}
}

func convertColumns(dbCols []database.Column) []generator.Column {
	cols := make([]generator.Column, len(dbCols))
	for i, c := range dbCols {
		cols[i] = generator.Column{
			Name:       c.Name,
			Type:       c.Type,
			IsNullable: c.IsNullable,
		}
	}
	return cols
}

func convertRelations(dbRels []database.Relation) []generator.Relation {
	rels := make([]generator.Relation, len(dbRels))
	for i, r := range dbRels {
		rels[i] = generator.Relation{
			ChildTable:   r.ChildTable,
			ChildColumn:  r.ChildColumn,
			ParentTable:  r.ParentTable,
			ParentColumn: r.ParentColumn,
		}
	}
	return rels
}

func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func SetupHealthCheck(app *fiber.App, db database.Database, logger interface{ Error(string, ...interface{}) }) {
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"database": fiber.Map{
					"status": "down",
				},
			})
		}

		return c.JSON(fiber.Map{
			"status": "healthy",
			"database": fiber.Map{
				"status": "up",
			},
		})
	})
}

func SetupOpenAPIUI(app *fiber.App) {
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
}
