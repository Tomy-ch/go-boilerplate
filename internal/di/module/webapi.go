package module

import (
	"go-boilerplate/internal/config"                                             // sample-api:line
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
		// sample-api:replace-begin
		fx.Provide(
			exchangerateext.NewEndpoint,
			provideCachedExchangeRateGateway,
			addressext.NewEndpoint,
			addressext.New,
		),
		provideHTTPClientProfiles(
			provideExchangeRateDownstreamProfile,
			addressext.NewDownstreamProfile,
		),
		provideRequiredDownstreams(
			exchangerateext.RequiredDownstream,
			addressext.RequiredDownstream,
		),
		// sample-api:replace-with
		// = // gateway を足すときは、コンストラクタを fx.Provide へ、HTTP クライアントの
		// = // プロファイルと必須 downstream をそれぞれのグループ提供子へ渡す。引数ゼロの
		// = // 呼び出しは fx.Options() を返すだけの no-op なので、雛形として残さない。
		// sample-api:replace-end
	)
}

// sample-api:begin

// provideCachedExchangeRateGateway は、素の為替レート gateway を TTL キャッシュ decorator で包み、
// usecase へ注入する boundary.Gateway を返します。ADR-0106 (no-generic-cache-abstraction) の decorator seam 原理を Gateway
// boundary へ適用し、TTL の時刻依存を infra 層 decorator に閉じます。
func provideCachedExchangeRateGateway(
	endpoint exchangerateext.Endpoint,
	client httpclient.Client,
	tf observability.TracerFactory,
	clk clock.Clock,
) exchangeratebd.Gateway {
	return exchangerateext.NewCache(exchangerateext.New(endpoint, client, tf), clk)
}

// sample-api:end

// sample-api:begin

// provideExchangeRateDownstreamProfile は、env に応じた SSRF ガード設定で為替レート取得プロファイルを構築します。
// private 網宛て接続の許可判定は allowPrivateNetworkForEnv に委ねます。DAST は疑似サービスを
// ランナー上に立てて叩くため、許可されない環境ではその経路に到達できません。
func provideExchangeRateDownstreamProfile(appCfg *config.ApplicationConfig) httpclient.DownstreamProfile {
	return exchangerateext.NewDownstreamProfile(allowPrivateNetworkForEnv(appCfg.Env()))
}

// sample-api:end
