package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"

	"github.com/getkin/kin-openapi/openapi3"
)

// validatorDeps は、ValidatorModule の解決に必要な設定依存を返します。
func validatorDeps(t *testing.T) fx.Option {
	t.Helper()

	return fx.Options(
		fx.Provide(func() testing.TB { return t }),
		fx.Provide(config.MockConfigForTest),
	)
}

func TestValidatorModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュールを組み込めば Validator が解決できる", func(t *testing.T) {
			t.Parallel()

			var v *openapi3.T

			validateGraph(t, validatorDeps(t), ValidatorModule(), fx.Populate(&v))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュール未配線では Validator が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var v *openapi3.T

			requireGraphIncomplete(t, validatorDeps(t), fx.Populate(&v))
		})
	})
}

func TestValidatorModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで Validator が提供される", func(t *testing.T) {
			t.Parallel()

			var v *openapi3.T
			app := fx.New(
				validatorDeps(t),
				ValidatorModule(),
				fx.Populate(&v),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, v)
		})
	})
}
