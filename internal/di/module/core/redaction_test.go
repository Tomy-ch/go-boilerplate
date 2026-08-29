package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/controller/httpstack/redaction"
)

func TestRedactionModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュールを組み込めば Redactor が解決できる", func(t *testing.T) {
			t.Parallel()

			var r redaction.Redactor

			validateGraph(t, validatorDeps(t), ValidatorModule(), RedactionModule(), fx.Populate(&r))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("モジュール未配線では Redactor が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var r redaction.Redactor

			requireGraphIncomplete(t, validatorDeps(t), ValidatorModule(), fx.Populate(&r))
		})
	})
}

func TestRedactionModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで実 spec の query apiKey を秘匿する Redactor が提供される", func(t *testing.T) {
			t.Parallel()

			var r redaction.Redactor
			app := fx.New(
				validatorDeps(t),
				ValidatorModule(),
				RedactionModule(),
				fx.Populate(&r),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.Equal(t, "/v1/streams/s?ticket="+redaction.RedactedValue, r.URI("/v1/streams/s?ticket=raw-value"))
		})
	})
}

func Test_provideRedactor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("spec の StreamTicket scheme から ticket を秘匿する Redactor を組み立てる", func(t *testing.T) {
			t.Parallel()

			spec, err := validator.GetValidator()
			require.NoError(t, err)
			r := provideRedactor(spec)
			assert.Equal(t, map[string][]string{"ticket": {redaction.RedactedValue}}, r.QueryParams(map[string][]string{"ticket": {"raw"}}))
		})
	})
}
