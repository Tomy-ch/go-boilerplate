package address_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/infrastructure/webapi/address"

	"github.com/stretchr/testify/assert"
)

func TestNewDownstreamProfile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("外部サービス向けにtrace非伝搬かつprivate拒否のプロファイルを返す", func(t *testing.T) {
			t.Parallel()

			dp := address.NewDownstreamProfile()
			assert.Equal(t, httpclient.Downstream("address"), dp.Name)
			assert.False(t, dp.Profile.PropagateTrace)
			assert.False(t, dp.Profile.AllowPrivateNetwork)
		})
	})
}

func TestRequiredDownstream(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本gatewayが使用するDownstreamを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, httpclient.Downstream("address"), address.RequiredDownstream())
		})
	})
}

func TestNewEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サンプル既定のEndpointを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, address.Endpoint("https://zipcloud.ibsnet.co.jp"), address.NewEndpoint())
		})
	})
}
