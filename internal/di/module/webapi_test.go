package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx" // sample-api:line

	"go-boilerplate/internal/infrastructure/httpclient"
	infrasystem "go-boilerplate/internal/infrastructure/system"                  // sample-api:line
	exchangerateext "go-boilerplate/internal/infrastructure/webapi/exchangerate" // sample-api:line
	"go-boilerplate/internal/observability"                                      // sample-api:line
	exchangeratebd "go-boilerplate/internal/usecase/boundary/exchangerate"       // sample-api:line
)

func Test_webapiModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 外部 Web API gateway は httpclient.Client / TracerFactory に依存するため、
	// substrate（clock + httpclient）も併せて配線し、グラフが欠落なく解決することを確認する。
	opts := append(commonDeps(), clockModule(), httpClientModule(), webapiModule())
	validateGraph(t, opts...)
}

// sample-api:begin
func Test_provideCachedExchangeRateGateway(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("素のGatewayをTTLキャッシュdecoratorで包んだGatewayを返す", func(t *testing.T) {
			t.Parallel()

			tf := observability.NewNoopTracerFactory(t)
			clk := infrasystem.NewClock()

			got := provideCachedExchangeRateGateway(exchangerateext.NewEndpoint(), nil, tf, clk)

			// 素の gateway ではなく TTL キャッシュ decorator で包まれていることを型で確認する。
			assert.IsType(t, exchangerateext.NewCache(nil, clk), got)
		})
	})
}

// sample-api:end

func Test_webapiModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言した required downstream には対応する profile が揃っている", func(t *testing.T) {
			t.Parallel()

			// required だけ寄与して profile を欠くと registry 構築が起動時に失敗する。
			// このモジュール自身の寄与が自己完結していることを value group の突き合わせで確認する。
			profiles := collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, webapiModule())
			required := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, webapiModule())

			assert.Empty(t, httpclient.MissingDownstreams(profiles, required))
		})

		// sample-api:begin
		t.Run("為替レートGatewayを提供する", func(t *testing.T) {
			t.Parallel()

			var gateway exchangeratebd.Gateway

			validateGraph(t, append(commonDeps(), clockModule(), httpClientModule(), webapiModule(),
				fx.Populate(&gateway))...)
		})
		// sample-api:end
	})
}
