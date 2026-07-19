package core

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	"go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/infrastructure/httpclient"

	// sample-api:replace-begin
	"go-boilerplate/internal/infrastructure/auth/useridentity"
	// sample-api:replace-with
	// = "go-boilerplate/internal/infrastructure/auth/identity"
	// sample-api:replace-end
	"go-boilerplate/internal/logging"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock"

	"go-boilerplate/pkg/xerrors"

	"go.uber.org/fx"
)

// callerSkipCount は、ロギングラッパーが追加するフレーム数を補正するためのスキップ数です。
const callerSkipCount = 1

// accessTokenType は、access token に期待する RFC 9068 の typ ヘッダ値です。
// mock 認証サーバーは access token に付与し ID Token には付与しないため、typ 不一致で ID Token 誤用を拒否します。
const accessTokenType = "at+jwt"

// authenticatorParams は、provideAuthenticator の依存を集約する fx パラメータです。
// HTTPClient は JWKS 取得に用いる outbound HTTP substrate で、infra 層（InfrastructureModule）が全プロファイルで提供する常設依存です。
type authenticatorParams struct {
	fx.In

	AppCfg     *config.ApplicationConfig
	AuthCfg    *config.AuthConfig
	Clock      clock.Clock
	Logger     logging.Logger
	HTTPClient httpclient.Client
}

// AuthnModule は、認証関連の依存関係（Authenticator・IdentityResolver・Auth コントローラ）を提供するfxモジュールを返します。
func AuthnModule() fx.Option {
	return fx.Module(
		"core.authn",
		fx.Provide(
			provideAuthenticator,
			// sample-api:replace-begin
			useridentity.New,
			// sample-api:replace-with
			// = identity.New,
			// sample-api:replace-end
			auth.NewAuthenticator,
		),
	)
}

// provideAuthenticator は、環境に対応した Authenticator を選んで返します。
// 対応する case が無い環境（staging / production 等）は誤った Authenticator を配線しないよう
// default で起動エラーにします（fail-closed）。
func provideAuthenticator(p authenticatorParams) (authbd.Authenticator, error) {
	logger := p.Logger.Named("core.authn").CallerSkip(callerSkipCount)

	switch p.AppCfg.Env() {
	case config.EnvCI, config.EnvTest:
		logger.Warn(
			context.Background(),
			"Local authenticator wired: authentication is stubbed (non-production only)",
			logging.String("env", p.AppCfg.Env()),
		)

		return local.New(), nil
	case config.EnvLocal, config.EnvDevelopment:
		return provideJWKSAuthenticator(p, logger)
	default:
		logger.Error(
			context.Background(),
			"No authenticator configured for the current environment",
			logging.String("env", p.AppCfg.Env()),
		)

		return nil, xerrors.New("no authenticator configured for environment: " + p.AppCfg.Env())
	}
}

// provideJWKSAuthenticator は、AUTH_* 設定から JWKS backed の JWT authenticator を構築します。
// JWKS の取得は httpclient substrate に委ねるため、goroutine / lifecycle の管理は不要です（遅延取得）。
func provideJWKSAuthenticator(p authenticatorParams, logger logging.Logger) (authbd.Authenticator, error) {
	authenticator, err := jwt.NewJWKS(jwt.JWKSParams{
		Params: jwt.Params{
			Issuer:       p.AuthCfg.Issuer(),
			Audience:     p.AuthCfg.Audience(),
			AllowedAlgs:  p.AuthCfg.AllowedAlgorithms(),
			Leeway:       p.AuthCfg.ClockSkew(),
			ExpectedType: accessTokenType,
			Clock:        p.Clock,
		},
		JWKSURL:  p.AuthCfg.JWKSURL(),
		CacheTTL: p.AuthCfg.JWKSCacheTTL(),
	}, p.HTTPClient)
	if err != nil {
		logger.Error(
			context.Background(),
			"Failed to wire JWKS JWT authenticator",
			logging.String("env", p.AppCfg.Env()),
			logging.Error("error", err),
		)

		return nil, err
	}

	logger.Info(
		context.Background(),
		"JWKS JWT authenticator wired",
		logging.String("env", p.AppCfg.Env()),
		logging.String("issuer", p.AuthCfg.Issuer()),
	)

	return authenticator, nil
}
