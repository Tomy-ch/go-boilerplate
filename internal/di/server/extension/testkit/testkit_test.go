package testkit_test

import (
	"testing"

	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

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

// runIsolated は、*testing.T を要求するヘルパー fn を隔離した throwaway な testing.T で実行し、fn 内の require が
// 失敗（FailNow → runtime.Goexit）したかどうかを返す。親テストへ失敗が伝播しないため、異常系の観測に用いる。
func runIsolated(fn func(t *testing.T)) bool {
	failed := make(chan bool, 1)
	go func() {
		completed := false
		defer func() { failed <- !completed }()

		var isolated testing.T
		fn(&isolated)
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

			failed := runIsolated(func(it *testing.T) {
				testkit.RequireProvidesOne[extension.PreMiddleware](it, "middlewares.pre",
					provideOnePre("only"),
				)
			})

			assert.False(t, failed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グループに要素が1件も provide されない場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated(func(it *testing.T) {
				testkit.RequireProvidesOne[extension.PreMiddleware](it, "middlewares.pre")
			})

			assert.True(t, failed)
		})

		t.Run("グループに要素が2件 provide される場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated(func(it *testing.T) {
				testkit.RequireProvidesOne[extension.PreMiddleware](it, "middlewares.pre",
					provideOnePre("first"),
					provideOnePre("second"),
				)
			})

			assert.True(t, failed)
		})

		t.Run("app.Start が依存不足で失敗する場合、検証が失敗する", func(t *testing.T) {
			t.Parallel()

			failed := runIsolated(func(it *testing.T) {
				testkit.RequireProvidesOne[extension.PreMiddleware](it, "middlewares.pre",
					fx.Provide(fx.Annotate(
						func(string) extension.PreMiddleware { return extension.PreMiddleware{} },
						fx.ResultTags(`group:"middlewares.pre"`),
					)),
				)
			})

			assert.True(t, failed)
		})
	})
}
