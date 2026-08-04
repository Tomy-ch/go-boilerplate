package module

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/observability"
)

// newAppCfgForEnv は、指定 env のアプリケーション設定を返します。
func newAppCfgForEnv(t *testing.T, env string) *config.ApplicationConfig {
	t.Helper()
	appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
	appCfg.SetApplicationEnv(t, env)

	return appCfg
}

func Test_allowPrivateNetworkForEnv(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local / CI / Test は private network を許可する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, allowPrivateNetworkForEnv(config.EnvLocal))
			assert.True(t, allowPrivateNetworkForEnv(config.EnvCI))
			assert.True(t, allowPrivateNetworkForEnv(config.EnvTest))
		})

		t.Run("dev / stg / prd および未知環境は許可しない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, allowPrivateNetworkForEnv(config.EnvDevelopment))
			assert.False(t, allowPrivateNetworkForEnv(config.EnvStaging))
			assert.False(t, allowPrivateNetworkForEnv(config.EnvProduction))
			assert.False(t, allowPrivateNetworkForEnv(""))
		})
	})
}

func Test_provideOutboundHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("env に応じて private 網宛ての可否が変わる", func(t *testing.T) {
			t.Parallel()
			// httptest は loopback（private）で待つ。計装なしの実 transport を使うのは、
			// テスト用の noop transport が全接続を許可する dial control を持ち、
			// 方針の違いが結果に出ないため。
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			// transport を共有すると接続プールも共有され、許可側が張った接続を拒否側が再利用して
			// dial 時のガードが発火しない。方針ごとに transport を分けて判定する。
			localErr := getWith(
				t,
				provideOutboundHTTPClient(observability.NewGuardedHTTPClientTransport(t), newAppCfgForEnv(t, config.EnvLocal)),
				srv.URL,
			)
			prdErr := getWith(
				t,
				provideOutboundHTTPClient(observability.NewGuardedHTTPClientTransport(t), newAppCfgForEnv(t, config.EnvProduction)),
				srv.URL,
			)

			require.NoError(t, localErr, "local で private 網宛てが拒否されている")
			require.Error(t, prdErr, "prd で private 網宛てが許可されており、env の方針が下流へ効いていない")
		})
	})
}

// getWith は、指定クライアントで url へ GET し、その結果のエラーだけを返します。
func getWith(t *testing.T, client *observability.OutboundHTTPClient, url string) error {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
