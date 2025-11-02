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

	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api"
	"github.com/nicolasbonnici/gorest/internal/logger"
	"github.com/nicolasbonnici/gorest/internal/middleware"
	"github.com/nicolasbonnici/gorest/pkg/database"
	_ "github.com/nicolasbonnici/gorest/pkg/database/mysql"
	_ "github.com/nicolasbonnici/gorest/pkg/database/postgres"
	_ "github.com/nicolasbonnici/gorest/pkg/database/sqlite"
)

type Config struct {
	DBDriver           string
	DBUrl              string
	JWTSecret          string
	JWTTTL             int
	Port               string
	PaginationLimit    int
	PaginationMaxLimit int
	CORSOrigins        string
}

func validateJWTSecretStrength(secret string) error {
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long (current: %d)", len(secret))
	}

	weakPatterns := []string{"secret", "password", "test", "admin", "123"}
	secretLower := strings.ToLower(secret)
	for _, pattern := range weakPatterns {
		if strings.Contains(secretLower, pattern) {
			return fmt.Errorf("JWT_SECRET contains common word '%s'. Generate secure secret: openssl rand -base64 32", pattern)
		}
	}

	uniqueChars := make(map[rune]bool)
	for _, c := range secret {
		uniqueChars[c] = true
	}
	if len(uniqueChars) < 16 {
		return fmt.Errorf("JWT_SECRET has low entropy (only %d unique characters). Use: openssl rand -base64 32", len(uniqueChars))
	}

	for i := 0; i < len(secret)-4; i++ {
		if secret[i] == secret[i+1] && secret[i] == secret[i+2] && secret[i] == secret[i+3] {
			return fmt.Errorf("JWT_SECRET has repeated characters pattern. Use: openssl rand -base64 32")
		}
	}

	return nil
}

func validateConfig(cfg Config) {
	if cfg.DBUrl == "" {
		logger.Log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if strings.Contains(cfg.DBUrl, "sslmode=disable") {
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			logger.Log.Error("❌ FATAL: Database SSL is DISABLED in production! This is a critical security violation.")
			os.Exit(1)
		}
		logger.Log.Warn("⚠️  Database SSL is DISABLED! This is insecure for production. Use sslmode=require or sslmode=verify-full")
	}

	if cfg.JWTSecret == "" {
		logger.Log.Error("JWT_SECRET is required")
		os.Exit(1)
	}

	if err := validateJWTSecretStrength(cfg.JWTSecret); err != nil {
		logger.Log.Error("JWT_SECRET validation failed", "error", err)
		os.Exit(1)
	}

	if cfg.JWTTTL <= 0 {
		logger.Log.Error("JWT_TTL must be a positive integer (seconds)", "current_value", cfg.JWTTTL)
		os.Exit(1)
	}

	if cfg.PaginationLimit <= 0 {
		logger.Log.Error("PAGINATION_LIMIT must be a positive integer", "current_value", cfg.PaginationLimit)
		os.Exit(1)
	}

	if cfg.PaginationMaxLimit <= 0 {
		logger.Log.Error("PAGINATION_MAX_LIMIT must be a positive integer", "current_value", cfg.PaginationMaxLimit)
		os.Exit(1)
	}

	if cfg.PaginationLimit > cfg.PaginationMaxLimit {
		logger.Log.Error("PAGINATION_LIMIT cannot exceed PAGINATION_MAX_LIMIT", "limit", cfg.PaginationLimit, "max", cfg.PaginationMaxLimit)
		os.Exit(1)
	}

	if cfg.Port == "" {
		logger.Log.Error("PORT is required")
		os.Exit(1)
	}
}

func Start(cfg Config) {
	validateConfig(cfg)
	projectRoot, err := internal.FindProjectRoot()
	if err != nil {
		logger.Log.Error("Failed to find project root", "error", err)
		os.Exit(1)
	}

	modelsDir := filepath.Join(projectRoot, "internal", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		logger.Log.Error("Models not found. Run 'make modelgen' first to generate models from database schema")
		os.Exit(1)
	}

	resourcesDir := filepath.Join(projectRoot, "internal", "api", "resources")
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		logger.Log.Error("Resources not found. Run 'make resourcegen' first to generate API resources")
		os.Exit(1)
	}

	openapiDir := filepath.Join(projectRoot, "internal", "openapi")
	if _, err := os.Stat(openapiDir); os.IsNotExist(err) {
		logger.Log.Error("OpenAPI schema not found. Run 'make openapigen' first to generate OpenAPI schema")
		os.Exit(1)
	}

	db, err := database.Open(cfg.DBDriver, cfg.DBUrl)
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

	tables := make(map[string]internal.TableSchema)
	for _, t := range schemaSlice {
		tables[t.TableName] = internal.TableSchema{
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

	app.Use(limiter.New(limiter.Config{
		Max:        50,
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

	corsOrigins := cfg.CORSOrigins
	if corsOrigins == "" {
		corsOrigins = "*"
		logger.Log.Warn("CORS_ORIGINS not set, defaulting to '*' (allow all). Set CORS_ORIGINS in production!")
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

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

	internal.SetupOpenAPIUI(app)
	internal.SetupHealthCheck(app, db)
	internal.SetupAuth(app, db, cfg.JWTSecret, cfg.JWTTTL)
	api.RegisterGeneratedRoutes(app, db, tables, cfg.JWTSecret, cfg.PaginationLimit, cfg.PaginationMaxLimit)

	internal.SetupOpenAPI(app, tables, cfg.PaginationLimit, cfg.PaginationMaxLimit)

	// Channel to listen for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logger.Log.Info("REST API running", "port", cfg.Port, "url", "http://localhost:"+cfg.Port)
		logger.Log.Info("Health check available", "url", "http://localhost:"+cfg.Port+"/health")
		if err := app.Listen(":" + cfg.Port); err != nil {
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

func convertColumns(dbCols []database.Column) []internal.Column {
	cols := make([]internal.Column, len(dbCols))
	for i, c := range dbCols {
		cols[i] = internal.Column{
			Name:       c.Name,
			Type:       c.Type,
			IsNullable: c.IsNullable,
		}
	}
	return cols
}

func convertRelations(dbRels []database.Relation) []internal.Relation {
	rels := make([]internal.Relation, len(dbRels))
	for i, r := range dbRels {
		rels[i] = internal.Relation{
			ChildTable:   r.ChildTable,
			ChildColumn:  r.ChildColumn,
			ParentTable:  r.ParentTable,
			ParentColumn: r.ParentColumn,
		}
	}
	return rels
}
