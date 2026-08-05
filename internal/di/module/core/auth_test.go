package core

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/local"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_observability "go-boilerplate/internal/observability/mock"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"
)

// newAuthParams は、指定した環境で provideAuthenticator を呼ぶための依存を組み立てます。
func newAuthParams(t *testing.T, env string, logger logging.Logger) authenticatorParams {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	appCfg.SetApplicationEnv(t, env)

	return authenticatorParams{
		AppCfg:  appCfg,
		AuthCfg: config.NewAuthConfig(cfg),
		Clock:   system.NewClock(),
		Logger:  logger,
	}
}

// authnDeps は、AuthnModule の解決に必要な設定・時刻・ログ・HTTPClient の依存を返します。
func authnDeps(t *testing.T) fx.Option {
	t.Helper()

	return fx.Options(
		fx.Provide(func() testing.TB { return t }),
		fx.Provide(func() *testing.T { return t }),
		fx.Provide(config.MockConfigForTest),
		fx.Provide(config.NewApplicationConfig),
		fx.Provide(config.NewAuthConfig),
		fx.Provide(system.NewClock),
		fx.Provide(logging.NewTestLogger),
		// HTTPClient は infra 層が常設提供する必須依存。test 環境ではスタブ認証で未使用だが、
		// authenticatorParams の解決に必要なためモックを供給する。
		fx.Provide(func() httpclient.Client {
			return mock_httpclient.NewMockClient(gomock.NewController(t))
		}),
	)
}

// resolverDeps は、useridentity.New が要求する RDB とトレーサの依存を返します。
// グラフ検証はコンストラクタを実行しないため、モックに期待呼び出しを設定する必要はありません。
func resolverDeps(t *testing.T) fx.Option {
	t.Helper()

	return fx.Options(
		fx.Provide(func() driver.DatabaseDriver {
			return mock_driver.NewMockDatabaseDriver(gomock.NewController(t))
		}),
		fx.Provide(func() observability.TracerFactory {
			return mock_observability.NewMockTracerFactory(gomock.NewController(t))
		}),
	)
}

func TestAuthnModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// AuthnModule が登録する 3 つの出力型をすべて要求する。fx は要求された型の依存しか解決しないため、
	// Authenticator だけを Populate すると IdentityResolver 側の配線が壊れても緑のままになる。
	populateAll := func() fx.Option {
		var (
			authenticator authbd.Authenticator
			resolver      authbd.IdentityResolver
			authFunc      openapi3filter.AuthenticationFunc
		)

		return fx.Populate(&authenticator, &resolver, &authFunc)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュールを組み込めば Authenticator と IdentityResolver と AuthenticationFunc が解決できる", func(t *testing.T) {
			t.Parallel()

			validateGraph(t, authnDeps(t), resolverDeps(t), AuthnModule(), populateAll())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュール未配線では Authenticator が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			requireGraphIncomplete(t, authnDeps(t), resolverDeps(t), populateAll())
		})
	})
}

func TestAuthnModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで環境に応じた Authenticator が提供される", func(t *testing.T) {
			t.Parallel()

			var a authbd.Authenticator
			app := fx.New(
				authnDeps(t),
				AuthnModule(),
				fx.Populate(&a),
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			// test 環境（MockConfigForTest 既定）では local.New() が配線される。
			assert.Equal(t, local.New(), a)
		})
	})
}

func Test_provideAuthenticator(t *testing.T) {
	t.Parallel()

	// staging / production はまだ実 IdP 未配線のため、default で fail-closed になること。
	assertFailClosed := func(t *testing.T, env string) {
		t.Helper()
		logger := logging.NewTestLogger(t)
		authenticator, err := provideAuthenticator(newAuthParams(t, env, logger))
		require.ErrorIs(t, err, errNoAuthenticatorForEnv)
		require.ErrorContains(t, err, env)
		assert.Nil(t, authenticator)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("CI環境では local.Authenticator が提供されWARNが出る", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvCI, logger))
			require.NoError(t, err)

			la := local.New()
			assert.Equal(t, la, authenticator)
			// スタブ配線時に WARN で注意喚起されること。
			assert.Len(t, logs.FilterMessage("Local authenticator wired: authentication is stubbed (non-production only)").All(), 1)
		})

		t.Run("テスト環境では local.Authenticator が提供される", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvTest, logger))
			require.NoError(t, err)

			la := local.New()
			assert.Equal(t, la, authenticator)
		})

		t.Run("local環境でAUTH設定とHTTPClientが揃えばJWKS authenticatorが返る", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			p := newAuthParams(t, config.EnvLocal, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "https://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideAuthenticator(p)
			require.NoError(t, err)
			assert.NotNil(t, authenticator)
		})

		t.Run("development環境でAUTH設定とHTTPClientが揃えばJWKS authenticatorが返る", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			p := newAuthParams(t, config.EnvDevelopment, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "https://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideAuthenticator(p)
			require.NoError(t, err)
			assert.NotNil(t, authenticator)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ローカル環境ではJWKS authenticatorを配線しAUTH未設定なら fail-closed になる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvLocal, logger))
			require.Error(t, err)
			assert.Nil(t, authenticator)
		})

		t.Run("development環境でAUTH設定が無い場合はfail-closedでエラーを返す", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			authenticator, err := provideAuthenticator(newAuthParams(t, config.EnvDevelopment, logger))
			require.Error(t, err)
			assert.Nil(t, authenticator)
		})

		t.Run("staging環境では Authenticator を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertFailClosed(t, config.EnvStaging)
		})

		t.Run("production環境では Authenticator を配線せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertFailClosed(t, config.EnvProduction)
		})
	})
}

func Test_provideJWKSAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("AUTH設定とHTTPClientが揃えばJWKS authenticatorを返しINFOを記録する", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			p := newAuthParams(t, config.EnvLocal, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "https://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideJWKSAuthenticator(p, logger)
			require.NoError(t, err)
			assert.NotNil(t, authenticator)
			assert.Len(t, logs.FilterMessage("JWKS JWT authenticator wired").All(), 1)
		})

		t.Run("local環境ではhttpのJWKS URLでも構築できる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			p := newAuthParams(t, config.EnvLocal, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "http://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideJWKSAuthenticator(p, logger)
			require.NoError(t, err)
			assert.NotNil(t, authenticator)
		})

		t.Run("development環境ではhttpsのJWKS URLで構築できる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			p := newAuthParams(t, config.EnvDevelopment, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "https://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideJWKSAuthenticator(p, logger)
			require.NoError(t, err)
			assert.NotNil(t, authenticator)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("issuerとJWKS URLの両方が未設定なら構築に失敗しERRORを記録する", func(t *testing.T) {
			t.Parallel()

			logger, logs := logging.NewObservedTestLogger(t)
			p := newAuthParams(t, config.EnvLocal, logger)
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideJWKSAuthenticator(p, logger)
			require.Error(t, err)
			assert.Nil(t, authenticator)
			assert.Len(t, logs.FilterMessage("Failed to wire JWKS JWT authenticator").All(), 1)
		})

		t.Run("development環境ではhttpのJWKS URLは設定エラーになる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			p := newAuthParams(t, config.EnvDevelopment, logger)
			p.AuthCfg.SetAuthIssuer(t, "https://issuer.example.com")
			p.AuthCfg.SetAuthAudience(t, "go-boilerplate-api")
			p.AuthCfg.SetAuthJWKSURL(t, "http://issuer.example.com/.well-known/jwks.json")
			p.HTTPClient = mock_httpclient.NewMockClient(gomock.NewController(t))

			authenticator, err := provideJWKSAuthenticator(p, logger)
			require.Error(t, err)
			assert.Nil(t, authenticator)
		})
	})
}
