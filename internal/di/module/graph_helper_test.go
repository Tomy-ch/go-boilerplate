package module

import (
	"testing"

	"go-boilerplate/internal/di/lifecycle"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// commonDeps は、各層モジュールのグラフ検証で共通して必要になる下位モジュール群を返す。
func commonDeps() []fx.Option {
	return []fx.Option{
		lifecycle.Module(),
		ConfigModule(),
		LoggingModule(),
		ObservabilityModule(),
		SystemModule(),
		DatabaseModule(),
	}
}

// validateGraph は、与えたモジュール群の依存グラフが欠落なく結線されることを検証する。
// fx.NopLogger は構成検証時のログ出力を抑えるだけで、検証結果（戻り値）には影響しない。
func validateGraph(t *testing.T, opts ...fx.Option) {
	t.Helper()
	require.NoError(t, fx.ValidateApp(append(opts, fx.NopLogger)...))
}

// collectGroup は、opt を組み込んだ fx アプリで value group tag に集まった値を返す。
// provide<X> ヘルパー群が「渡したコンストラクタをそのグループへ登録する」ことの検証に使う。
func collectGroup[T any](t *testing.T, tag string, opt fx.Option) []T {
	t.Helper()
	var got []T
	app := fx.New(
		opt,
		fx.Invoke(fx.Annotate(func(vs []T) { got = vs }, fx.ParamTags(tag))),
		fx.NopLogger,
	)
	require.NoError(t, app.Err())
	return got
}
