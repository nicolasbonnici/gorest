package main

import (
	"example.com/basic-api/generated/resources"
	customplugins "example.com/basic-api/plugins"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"

	codegenPlugin "github.com/nicolasbonnici/gorest-codegen"
	openapiplugin "github.com/nicolasbonnici/gorest-openapi"
	statusplugin "github.com/nicolasbonnici/gorest-status"
)

func init() {
	pluginloader.RegisterPluginFactory("status", statusplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("codegen", codegenPlugin.NewPlugin)
	pluginloader.RegisterPluginFactory("openapi", openapiplugin.NewPlugin)

	pluginloader.RegisterPluginFactory("dummy", customplugins.NewDummyPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
