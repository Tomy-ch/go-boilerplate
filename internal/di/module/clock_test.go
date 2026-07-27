package module

import (
	"testing"
)

func Test_clockModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 時刻・待機（Clock / Sleeper）の配線のみを検証する。実挙動は infra 層のテストに任せる。
	opts := append(commonDeps(), clockModule())
	validateGraph(t, opts...)
}

func Test_clockModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
