package module

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"

	"go.uber.org/fx"
)

// authModule は、認証インフラ（JWT/JWKS）が使う httpclient substrate の Downstream プロファイルを提供する fx.Module です。
// JWKS 取得の substrate 設定（timeout / retry / breaker / budget）のみを他 gateway と同じvalue group 経由で登録します。
func authModule() fx.Option {
	return fx.Module("auth",
		provideHTTPClientProfiles(
			provideJWKSDownstreamProfile,
		),
		provideRequiredDownstreams(
			jwt.RequiredDownstream,
		),
	)
}

// provideJWKSDownstreamProfile は、env に応じた SSRF ガード設定で JWKS/discovery 取得プロファイルを構築します。
// private 網宛て接続の許可判定は allowPrivateNetworkForEnv に委ねます。
func provideJWKSDownstreamProfile(appCfg *config.ApplicationConfig) httpclient.DownstreamProfile {
	return jwt.NewDownstreamProfile(allowPrivateNetworkForEnv(appCfg.Env()))
}
