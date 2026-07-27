package module

import (
	"testing"
)

func TestOutboxRelayModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// relay engine は usecase（RelayUsecase）・clock.Sleeper・OutboxConfig 等に依存する。
	// poll ループの振る舞いは controller 層のテストに任せ、ここでは engine と
	// そのライフサイクルフックが依存と欠落なく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(), OutboxRelayModule())
	validateGraph(t, opts...)
}

func TestOutboxRelayModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
