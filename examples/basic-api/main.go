package main

import (
	"example.com/basic-api/generated/resources"
	customplugins "example.com/basic-api/plugins"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"

	authplugin "github.com/nicolasbonnici/gorest-auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	healthplugin "github.com/nicolasbonnici/gorest/plugins/health"
	openapiplugin "github.com/nicolasbonnici/gorest/plugins/openapi"
)

func init() {
	pluginloader.RegisterPluginFactory("contenttype", contenttypeplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("health", healthplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
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
