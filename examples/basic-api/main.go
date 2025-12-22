package main

import (
	"example.com/basic-api/generated/resources"
	customplugins "example.com/basic-api/plugins"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"

	authplugin "github.com/nicolasbonnici/gorest-auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"
	healthplugin "github.com/nicolasbonnici/gorest/plugins/health"
	loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
	openapiplugin "github.com/nicolasbonnici/gorest/plugins/openapi"
	ratelimitplugin "github.com/nicolasbonnici/gorest/plugins/ratelimit"
	requestidplugin "github.com/nicolasbonnici/gorest/plugins/requestid"
)

func init() {
	// Register built-in plugins
	pluginloader.RegisterPluginFactory("requestid", requestidplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("logger", loggerplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("cors", corsplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("ratelimit", ratelimitplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("contenttype", contenttypeplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("health", healthplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("openapi", openapiplugin.NewPlugin)

	// Register custom plugins
	pluginloader.RegisterPluginFactory("dummy", customplugins.NewDummyPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
