package module

import (
	"go-boilerplate/internal/infrastructure/httpclient"                          // sample-api:line
	addressext "go-boilerplate/internal/infrastructure/webapi/address"           // sample-api:line
	exchangerateext "go-boilerplate/internal/infrastructure/webapi/exchangerate" // sample-api:line
	"go-boilerplate/internal/observability"                                      // sample-api:line
	"go-boilerplate/internal/usecase/boundary/clock"                             // sample-api:line
	exchangeratebd "go-boilerplate/internal/usecase/boundary/exchangerate"       // sample-api:line

	"go.uber.org/fx"
)

// webapiModule は、外部 Web API クライアント（gateway）を提供するfx.Moduleです。
func webapiModule() fx.Option {
	return fx.Module("webapi",
		fx.Provide(
			exchangerateext.NewEndpoint,      // sample-api:line
			provideCachedExchangeRateGateway, // sample-api:line
			addressext.NewEndpoint,           // sample-api:line
			addressext.New,                   // sample-api:line
		),
		provideHTTPClientProfiles(
			exchangerateext.NewDownstreamProfile, // sample-api:line
			addressext.NewDownstreamProfile,      // sample-api:line
		),
		provideRequiredDownstreams(
			exchangerateext.RequiredDownstream, // sample-api:line
			addressext.RequiredDownstream,      // sample-api:line
		),
	)
}

// provideCachedExchangeRateGateway は、素の為替レート gateway を TTL キャッシュ decorator で包み、
// usecase へ注入する boundary.Gateway を返します。ADR-0103 の decorator seam 原理を Gateway
// boundary へ適用し、TTL の時刻依存を infra 層 decorator に閉じます。
// sample-api:begin
func provideCachedExchangeRateGateway(
	endpoint exchangerateext.Endpoint,
	client httpclient.Client,
	tf observability.TracerFactory,
	clk clock.Clock,
) exchangeratebd.Gateway {
	return exchangerateext.NewCache(exchangerateext.New(endpoint, client, tf), clk)
}

// sample-api:end
