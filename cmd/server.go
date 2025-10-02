package main

import (
	"os"

	"github.com/tonmodule/restgen/pkg"
)

func main() {
	cfg := restgen.Config{
		DBUrl:     "postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable",
		JWTSecret: os.Getenv("JWT_SECRET"),
		Port:      "3000",
	}
	restgen.Start(cfg)
}
