package di

import (
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/controller/httpstack/banner"
	"boilerplate-go/internal/controller/httpstack/binder"
	"boilerplate-go/internal/controller/httpstack/cookie"
	"boilerplate-go/internal/controller/httpstack/cors"
	"boilerplate-go/internal/controller/httpstack/debugmode"
	"boilerplate-go/internal/controller/httpstack/defaultport"
	"boilerplate-go/internal/controller/httpstack/errorhandler"
	"boilerplate-go/internal/controller/httpstack/forcejson"
	"boilerplate-go/internal/controller/httpstack/ipextractor"
	"boilerplate-go/internal/controller/httpstack/logging"
	"boilerplate-go/internal/controller/httpstack/observability"
	"boilerplate-go/internal/controller/httpstack/recovery"
	"boilerplate-go/internal/controller/httpstack/requestid"
	"boilerplate-go/internal/controller/httpstack/security"
	"boilerplate-go/internal/controller/httpstack/uri"
	"boilerplate-go/internal/controller/httpstack/validator"

	"go.uber.org/fx"
)

// HTTPStackModule は、HTTP スタック関連の依存関係を提供するfx.Moduleです。
func HTTPStackModule() fx.Option {
	return fx.Module("httpstack",
		// Core Modules
		validator.CoreModule(),
		// Middleware Modules
		banner.Module(),
		binder.Module(),
		cookie.Module(),
		cors.Module(),
		debugmode.Module(),
		defaultport.Module(),
		errorhandler.Module(),
		forcejson.Module(),
		ipextractor.Module(),
		logging.Module(),
		recovery.Module(),
		requestid.Module(),
		security.Module(),
		uri.Module(),
		validator.Module(),
		observability.Module(),
		fx.Provide(
			httpstack.ApplyExtends,
		),
	)
}
