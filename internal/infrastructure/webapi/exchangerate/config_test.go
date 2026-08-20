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

		t.Run("外部サービス向けにtrace非伝搬のプロファイルを返す", func(t *testing.T) {
			t.Parallel()

			dp := exchangerate.NewDownstreamProfile(false)
			assert.Equal(t, httpclient.Downstream("exchangerate"), dp.Name)
			assert.False(t, dp.Profile.PropagateTrace)
			assert.False(t, dp.Profile.AllowPrivateNetwork)
		})

		t.Run("private網を許可する環境ではその指定がプロファイルへ反映される", func(t *testing.T) {
			t.Parallel()

			dp := exchangerate.NewDownstreamProfile(true)
			assert.True(t, dp.Profile.AllowPrivateNetwork)
			assert.False(t, dp.Profile.PropagateTrace)
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
