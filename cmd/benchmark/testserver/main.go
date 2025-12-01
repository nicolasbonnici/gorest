package main

import (
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"
	"github.com/nicolasbonnici/gorest/test/generated/resources"

	authplugin "github.com/nicolasbonnici/gorest/plugins/auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"
	healthplugin "github.com/nicolasbonnici/gorest/plugins/health"
	loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
	ratelimitplugin "github.com/nicolasbonnici/gorest/plugins/ratelimit"
	requestidplugin "github.com/nicolasbonnici/gorest/plugins/requestid"
	securityplugin "github.com/nicolasbonnici/gorest/plugins/security"
)

func init() {
	pluginloader.RegisterPluginFactory("requestid", requestidplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("logger", loggerplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("cors", corsplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("ratelimit", ratelimitplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("security", securityplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("contenttype", contenttypeplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
	pluginloader.RegisterPluginFactory("health", healthplugin.NewPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     "./cmd/benchmark/testserver",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
