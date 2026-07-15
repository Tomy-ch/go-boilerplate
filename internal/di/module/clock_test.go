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
