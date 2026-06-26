package module

import (
	exchangerateext "go-boilerplate/internal/infrastructure/webapi/exchangerate" // sample-api:line

	"go.uber.org/fx"
)

// webapiModule は、外部 Web API クライアント（gateway）を提供するfx.Moduleです。
func webapiModule() fx.Option {
	return fx.Module("webapi",
		fx.Provide(
			// サンプルの外部サービス gateway（DTO モード）
			exchangerateext.NewEndpoint, // sample-api:line
			exchangerateext.New,         // sample-api:line
		),
		provideHTTPClientProfiles(
			exchangerateext.NewDownstreamProfile, // sample-api:line
		),
	)
}
