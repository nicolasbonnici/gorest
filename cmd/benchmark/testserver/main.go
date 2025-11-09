package main

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/test/generated/resources"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Log.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-secret"
	}

	jwtTTL := 900
	if ttlStr := os.Getenv("JWT_TTL"); ttlStr != "" {
		parsedTTL, err := strconv.Atoi(ttlStr)
		if err == nil {
			jwtTTL = parsedTTL
		}
	}

	paginationLimit := 100
	if limitStr := os.Getenv("PAGINATION_LIMIT"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil {
			paginationLimit = parsedLimit
		}
	}

	paginationMaxLimit := 1000
	if maxStr := os.Getenv("PAGINATION_MAX_LIMIT"); maxStr != "" {
		parsedMax, err := strconv.Atoi(maxStr)
		if err == nil {
			paginationMaxLimit = parsedMax
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	corsOrigins := os.Getenv("CORS_ORIGINS")

	cfg := gorest.Config{
		DBUrl:              dbURL,
		JWTSecret:          jwtSecret,
		JWTTTL:             jwtTTL,
		Port:               port,
		PaginationLimit:    paginationLimit,
		PaginationMaxLimit: paginationMaxLimit,
		CORSOrigins:        corsOrigins,
		RegisterRoutes:     resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
