// Package outbound は、サーバーの応答や出力時の拡張機能に関するDIモジュールを提供します。
package outbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v5"
	"go.uber.org/fx"
)

// ErrorHandlerModule は、Error Handler モジュールを提供します。
func ErrorHandlerModule() fx.Option {
	return fx.Module("server.errorhandler",
		fx.Provide(
			provideDetailPolicy,
			provideAllowPolicy,
			provideErrorHandlerServeConfig,
		),
	)
}

// provideDetailPolicy は、OpenAPI spec から [errorhandler.DetailPolicy] を構築して fx に提供します。
// spec の解析に失敗した場合はエラーを返し、fx がアプリ起動を中断します。
func provideDetailPolicy(spec *openapi3.T) (errorhandler.DetailPolicy, error) {
	return errorhandler.NewOpenAPIDetailPolicy(spec)
}

// provideAllowPolicy は、OpenAPI spec から [errorhandler.AllowPolicy] を構築して fx に提供します。
// spec の解析に失敗した場合はエラーを返し、fx がアプリ起動を中断します。
func provideAllowPolicy(spec *openapi3.T) (errorhandler.AllowPolicy, error) {
	return errorhandler.NewOpenAPIAllowPolicy(spec)
}

// provideErrorHandlerServeConfig は、Error Handler のサーバー設定を提供します。
func provideErrorHandlerServeConfig(
	detailPolicy errorhandler.DetailPolicy,
	allowPolicy errorhandler.AllowPolicy,
	log logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig,
) extension.ServeCfgOut {
	policies := errorhandler.Policies{Detail: detailPolicy, Allow: allowPolicy}

	return extension.ServeCfgOut{
		SrvCfg: extension.SrvCfg{
			Name: "errorhandler",
			Config: func(e *echo.Echo) {
				errorhandler.New(e, policies, log, lf, obsCfg)
			},
		},
	}
}
