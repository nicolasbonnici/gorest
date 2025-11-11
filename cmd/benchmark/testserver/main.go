package main

import (
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/test/generated/resources"
)

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
