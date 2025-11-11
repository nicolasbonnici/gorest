package main

import (
	"example.com/basic-api/generated/resources"
	"github.com/nicolasbonnici/gorest"
)

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
