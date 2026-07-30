package core

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// validateGraph は、与えたモジュール群の依存グラフが欠落なく結線されることを検証する。
// fx.NopLogger は構成検証時のログ出力を抑えるだけで、検証結果（戻り値）には影響しない。
func validateGraph(t *testing.T, opts ...fx.Option) {
	t.Helper()
	require.NoError(t, fx.ValidateApp(append(opts, fx.NopLogger)...))
}

// requireGraphIncomplete は、与えたモジュール群では要求された型を解決できないことを検証する。
// 対象モジュールを外した構成に用いることで、そのモジュールこそが当該型の供給元であることを示す。
func requireGraphIncomplete(t *testing.T, opts ...fx.Option) {
	t.Helper()
	require.Error(t, fx.ValidateApp(append(opts, fx.NopLogger)...))
}
