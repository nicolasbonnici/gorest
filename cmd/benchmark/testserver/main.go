package main

import (
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"
	"github.com/nicolasbonnici/gorest/test/generated/resources"

	authplugin "github.com/nicolasbonnici/gorest/plugins/auth"
	contenttypeplugin "github.com/nicolasbonnici/gorest/plugins/contenttype"
	corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"
	loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
	ratelimitplugin "github.com/nicolasbonnici/gorest/plugins/ratelimit"
	requestidplugin "github.com/nicolasbonnici/gorest/plugins/requestid"
	securityplugin "github.com/nicolasbonnici/gorest/plugins/security"
)

func init() {
	pluginloader.RegisterGlobalPluginFactory("requestid", requestidplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("logger", loggerplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("cors", corsplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("ratelimit", ratelimitplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("security", securityplugin.NewPlugin)
	pluginloader.RegisterGlobalPluginFactory("contenttype", contenttypeplugin.NewPlugin)
	pluginloader.RegisterRoutePluginFactory("auth", authplugin.NewPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath:     "./cmd/benchmark/testserver",
		RegisterRoutes: resources.RegisterGeneratedRoutes,
	}

	gorest.Start(cfg)
}
