package main

import (
	"example.com/basic-api/generated/resources"
	customplugins "example.com/basic-api/plugins"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"

	authplugin "github.com/nicolasbonnici/gorest/plugins/auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"
	loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
	ratelimitplugin "github.com/nicolasbonnici/gorest/plugins/ratelimit"
	requestidplugin "github.com/nicolasbonnici/gorest/plugins/requestid"
	securityplugin "github.com/nicolasbonnici/gorest/plugins/security"
)

func init() {
	// Register built-in plugins
	pluginloader.RegisterGlobalPluginFactory("requestid", requestidplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("logger", loggerplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("cors", corsplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("ratelimit", ratelimitplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("security", securityplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("contenttype", contenttypeplugin.NewPlugin)
	pluginloader.RegisterRoutePluginFactory("auth", authplugin.NewPlugin)

	// Register custom plugins
	pluginloader.RegisterGlobalPluginFactory("dummy", customplugins.NewDummyPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     ".",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
