package module

import (
	"go.uber.org/fx"
)

// webapiModule は、外部 Web API クライアント（gateway）を提供するfx.Moduleです。
func webapiModule() fx.Option {
	return fx.Module("webapi",
		fx.Provide(),
		provideHTTPClientProfiles(),
		provideRequiredDownstreams(),
	)
}
