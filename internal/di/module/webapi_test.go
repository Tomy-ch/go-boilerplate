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
