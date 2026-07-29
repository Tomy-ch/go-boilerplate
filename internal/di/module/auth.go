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
// env→bool の解決を DI で行い（infra は env を知らない）、疑似 provider（private hostname）へ到達する
// local / CI / Test のみ private 網宛て接続を許可します。
func provideJWKSDownstreamProfile(appCfg *config.ApplicationConfig) httpclient.DownstreamProfile {
	return jwt.NewDownstreamProfile(allowPrivateNetworkForJWKSEnv(appCfg.Env()))
}

// allowPrivateNetworkForJWKSEnv は、JWKS/discovery 取得で private 網宛て接続を許可してよい環境かを返します。
// 実 IdP（public）へ接続する dev / stg / prd では false（SSRF 遮断）にします。
func allowPrivateNetworkForJWKSEnv(env string) bool {
	switch env {
	case config.EnvLocal, config.EnvCI, config.EnvTest:
		return true
	default:
		return false
	}
}
