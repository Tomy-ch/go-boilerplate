package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/usecase/boundary/clock"
)

func Test_clockModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 時刻・待機（Clock / Sleeper）の配線のみを検証する。実挙動は infra 層のテストに任せる。
	opts := append(commonDeps(), clockModule())
	validateGraph(t, opts...)
}

func Test_clockModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Clock と Sleeper を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				clk clock.Clock
				slp clock.Sleeper
			)

			validateGraph(t, append(commonDeps(), clockModule(), fx.Populate(&clk, &slp))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Clock が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var clk clock.Clock

			opts := append(commonDeps(), fx.Populate(&clk), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
