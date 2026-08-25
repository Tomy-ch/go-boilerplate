package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/httpclient"
)

func Test_webapiModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 外部 Web API gateway は httpclient.Client / TracerFactory に依存するため、
	// substrate（clock + httpclient）も併せて配線し、グラフが欠落なく解決することを確認する。
	opts := append(commonDeps(), clockModule(), httpClientModule(), webapiModule())
	validateGraph(t, opts...)
}

func Test_webapiModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言した required downstream には対応する profile が揃っている", func(t *testing.T) {
			t.Parallel()

			// required だけ寄与して profile を欠くと registry 構築が起動時に失敗する。
			// このモジュール自身の寄与が自己完結していることを value group の突き合わせで確認する。
			// profile の一部は env から SSRF ガードを決めるため、設定も併せて配線する。
			deps := func() fx.Option {
				return fx.Options(webapiModule(), fx.Provide(func() *config.ApplicationConfig {
					return config.NewApplicationConfig(config.MockConfigForTest(t))
				}))
			}
			profiles := collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, deps())
			required := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, deps())

			assert.Empty(t, httpclient.MissingDownstreams(profiles, required))
		})
	})
}
