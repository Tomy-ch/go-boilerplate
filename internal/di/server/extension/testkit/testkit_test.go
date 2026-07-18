package testkit_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

// provideOnePre は、middlewares.pre グループへ PreMiddleware をちょうど1件 provide するモジュールを返す。
func provideOnePre(name string) fx.Option {
	return fx.Provide(fx.Annotate(
		func() extension.PreMiddleware { return extension.PreMiddleware{Name: name} },
		fx.ResultTags(`group:"middlewares.pre"`),
	))
}

// provideStopError は、OnStop で必ずエラーを返す fx モジュールを返す。
func provideStopError() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStop: func(context.Context) error { return xerrors.New("stop failure") },
		})
	})
}

// runIsolated は RequireProvidesOne を隔離実行し、内部の require 失敗を親テストへ伝播させずに捕捉して返す（異常系の観測用）。
func runIsolated[T any](group string, opts ...fx.Option) bool {
	failed := make(chan bool, 1)
	go func() {
		completed := false
		defer func() { failed <- !completed }()

		var isolated testing.T
		testkit.RequireProvidesOne[T](&isolated, group, opts...)
		completed = true
	}()

	return <-failed
}

func TestRequireProvidesOne(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グループに要素が1件だけ provide される場合、検証を通過する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated[extension.PreMiddleware]("middlewares.pre",
				provideOnePre("only"),
			)

			assert.False(t, failed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グループに要素が1件も provide されない場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated[extension.PreMiddleware]("middlewares.pre")

			assert.True(t, failed)
		})

		t.Run("グループに要素が2件 provide される場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated[extension.PreMiddleware]("middlewares.pre",
				provideOnePre("first"),
				provideOnePre("second"),
			)

			assert.True(t, failed)
		})

		t.Run("app.Start が依存不足で失敗する場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated[extension.PreMiddleware]("middlewares.pre",
				fx.Provide(fx.Annotate(
					func(string) extension.PreMiddleware { return extension.PreMiddleware{} },
					fx.ResultTags(`group:"middlewares.pre"`),
				)),
			)

			assert.True(t, failed)
		})

		t.Run("app.Stop が失敗する場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			// provide 件数は 1 で Start/Len は通過し、OnStop エラーで app.Stop のみ失敗する経路。
			failed := runIsolated[extension.PreMiddleware]("middlewares.pre",
				provideOnePre("only"),
				provideStopError(),
			)

			assert.True(t, failed)
		})
	})
}
