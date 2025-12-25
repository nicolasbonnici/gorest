package main

import (
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"
	"github.com/nicolasbonnici/gorest/test/generated/resources"

	authplugin "github.com/nicolasbonnici/gorest-auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	healthplugin "github.com/nicolasbonnici/gorest/plugins/health"
	ratelimitplugin "github.com/nicolasbonnici/gorest/plugins/ratelimit"
)

func init() {
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
