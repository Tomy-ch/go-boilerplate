package module

import (
	"testing"
)

func Test_outboxPublisherModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// outbox publish 先（HTTP）は httpclient.Client / TracerFactory に依存するため、
	// substrate（clock + httpclient）も併せて配線し、グラフが欠落なく解決することを確認する。
	opts := append(commonDeps(), clockModule(), httpClientModule(), outboxPublisherModule())
	validateGraph(t, opts...)
}

func Test_outboxPublisherModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
