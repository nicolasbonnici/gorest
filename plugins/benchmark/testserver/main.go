package main

import (
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"
	"github.com/nicolasbonnici/gorest/test/generated/resources"

	authplugin "github.com/nicolasbonnici/gorest-auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"
	healthplugin "github.com/nicolasbonnici/gorest/plugins/health"
	loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
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
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
