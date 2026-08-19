package exchangerate_test

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/infrastructure/webapi/exchangerate"

	"github.com/stretchr/testify/assert"
)

func TestNewDownstreamProfile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("外部サービス向けにtrace非伝搬かつprivate拒否のプロファイルを返す", func(t *testing.T) {
			t.Parallel()

			dp := exchangerate.NewDownstreamProfile()
			assert.Equal(t, httpclient.Downstream("exchangerate"), dp.Name)
			assert.False(t, dp.Profile.PropagateTrace)
			assert.False(t, dp.Profile.AllowPrivateNetwork)
		})
	})
}

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定のベースURLをEndpointとして返す", func(t *testing.T) {
			t.Parallel()
			epCfg := config.NewEndpointConfig(config.MockConfigForTest(t))
			assert.Equal(t, exchangerate.Endpoint(epCfg.ExchangeRate()), exchangerate.NewEndpoint(epCfg))
		})
	})
}
