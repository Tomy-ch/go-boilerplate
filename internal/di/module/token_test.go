package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/usecase/boundary/token"
)

func Test_tokenModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// トークン生成（Generator）の配線のみを検証する。実挙動は infra 層のテストに任せる。
	opts := append(commonDeps(), tokenModule())
	validateGraph(t, opts...)
}

func Test_tokenModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Generator を提供する", func(t *testing.T) {
			t.Parallel()

			var gen token.Generator

			validateGraph(t, append(commonDeps(), tokenModule(), fx.Populate(&gen))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Generator が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var gen token.Generator

			opts := append(commonDeps(), fx.Populate(&gen), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
