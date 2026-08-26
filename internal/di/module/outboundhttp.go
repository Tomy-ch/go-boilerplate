package module

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/observability"
)

// provideOutboundHTTPClient は、外部 SDK へ渡す SSRF ガード付き HTTP クライアントを構築します。
// 自前の substrate を通せない SDK（AWS SDK 等）はこれを受け取ります。
func provideOutboundHTTPClient(
	transport *observability.HTTPClientTransport, appCfg *config.ApplicationConfig,
) *observability.OutboundHTTPClient {
	return observability.NewOutboundHTTPClient(transport, allowPrivateNetworkForEnv(appCfg.Env()))
}

// allowPrivateNetworkForEnv は、private 網宛て接続を許可してよい環境かを返します。
// local / CI / Test は compose 内のサービス（private アドレス）へ接続するため許可し、実サービスへ
// 接続する dev / stg / prd では false にします（SSRF 遮断）。dast も localhost の mock 認証サーバーから
// JWKS を取得するため許可します。env→bool の解決を DI で行うのは、infra が env を知らないためです。
func allowPrivateNetworkForEnv(env string) bool {
	switch env {
	case config.EnvLocal, config.EnvCI, config.EnvTest, config.EnvDast:
		return true
	default:
		return false
	}
}
