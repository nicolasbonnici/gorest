package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/nicolasbonnici/gorest/pkg"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	cfg := gorest.Config{
		DBUrl:     dbURL,
		JWTSecret: jwtSecret,
		Port:      port,
	}

	gorest.Start(cfg)
}
