package main

import (
	"os"

	"github.com/nicolasbonnici/gorest/pkg"
)

func main() {
	cfg := gorest.Config{
		DBUrl:     "postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable",
		JWTSecret: os.Getenv("JWT_SECRET"),
		Port:      "3000",
	}
	gorest.Start(cfg)
}
