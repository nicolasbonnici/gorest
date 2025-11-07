package main

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/nicolasbonnici/gorest/internal/logger"
	"github.com/nicolasbonnici/gorest/pkg"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Log.Info("No .env file found, using environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Log.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Log.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	jwtTTL := 900
	if ttlStr := os.Getenv("JWT_TTL"); ttlStr != "" {
		parsedTTL, err := strconv.Atoi(ttlStr)
		if err != nil {
			logger.Log.Error("JWT_TTL must be a valid integer", "error", err)
			os.Exit(1)
		}
		jwtTTL = parsedTTL
	}

	paginationLimit := 100
	if limitStr := os.Getenv("PAGINATION_LIMIT"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			logger.Log.Error("PAGINATION_LIMIT must be a valid integer", "error", err)
			os.Exit(1)
		}
		paginationLimit = parsedLimit
	}

	paginationMaxLimit := 1000
	if maxStr := os.Getenv("PAGINATION_MAX_LIMIT"); maxStr != "" {
		parsedMax, err := strconv.Atoi(maxStr)
		if err != nil {
			logger.Log.Error("PAGINATION_MAX_LIMIT must be a valid integer", "error", err)
			os.Exit(1)
		}
		paginationMaxLimit = parsedMax
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
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
	}

	gorest.Start(cfg)
}
