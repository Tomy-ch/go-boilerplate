package module

import (
	"testing"
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
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

// sample-api:end

func Test_webapiModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
