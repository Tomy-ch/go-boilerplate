package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"
)

// authModule は jwks の Downstream プロファイルと required downstream を value group へ寄与する。
// registry は起動時に required に対応する profile の充足を検証するため、fx.ValidateApp（結線のみ）では
// この寄与を検証できない。実アプリを起動し、jwks required が authModule の profile で充足されて
// httpclient.Client が構築されることを確認する。
//
//nolint:paralleltest // httpclient_test と同様、fx アプリ起動は global registerer 競合回避のため非並列
func Test_authModule_ProvidesJWKSProfile(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("jwks required に対応する profile が authModule で揃い Client が起動する", func(t *testing.T) {
			var client httpclient.Client
			app := newHTTPClientTestApp(t, authModule(), fx.Populate(&client))

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, client)
		})
	})
}

func Test_provideJWKSDownstreamProfile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非本番 env では private network を許可する profile を構築する", func(t *testing.T) {
			t.Parallel()
			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			assert.True(t, provideJWKSDownstreamProfile(appCfg).Profile.AllowPrivateNetwork)
		})

		t.Run("それ以外の env では private network を許可しない profile を構築する", func(t *testing.T) {
			t.Parallel()
			appCfg := config.NewApplicationConfig(&config.Config{})
			assert.False(t, provideJWKSDownstreamProfile(appCfg).Profile.AllowPrivateNetwork)
		})
	})
}

func Test_authModule(t *testing.T) {
	t.Parallel()

	// profile コンストラクタが ApplicationConfig を要求するため、value group の収集にも設定を供給する。
	moduleWithConfig := func(t *testing.T) fx.Option {
		t.Helper()
		return fx.Options(
			authModule(),
			fx.Provide(func() *config.ApplicationConfig {
				return config.NewApplicationConfig(config.MockConfigForTest(t))
			}),
		)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKS 取得を required downstream として宣言する", func(t *testing.T) {
			t.Parallel()

			required := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, moduleWithConfig(t))

			assert.Equal(t, []httpclient.Downstream{jwt.RequiredDownstream()}, required)
		})

		t.Run("宣言した required downstream には対応する profile が揃っている", func(t *testing.T) {
			t.Parallel()

			// required だけ寄与して profile を欠くと registry 構築が起動時に失敗する。
			profiles := collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, moduleWithConfig(t))
			required := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, moduleWithConfig(t))

			assert.Empty(t, httpclient.MissingDownstreams(profiles, required))
		})
	})
}
