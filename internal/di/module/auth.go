package module

import (
	"go-boilerplate/internal/infrastructure/auth/jwt"

	"go.uber.org/fx"
)

// authModule は、認証インフラ（JWT/JWKS）が使う httpclient substrate の Downstream プロファイルを提供する fx.Module です。
// JWKS 取得の substrate 設定（timeout / retry / breaker / budget）のみを他 gateway と同じvalue group 経由で登録します。
func authModule() fx.Option {
	return fx.Module("auth",
		provideHTTPClientProfiles(
			jwt.NewDownstreamProfile,
		),
		provideRequiredDownstreams(
			jwt.RequiredDownstream,
		),
	)
}
