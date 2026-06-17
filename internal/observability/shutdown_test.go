package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// errSpanExporter は Shutdown で任意のエラーを返す SpanExporter スタブ。
// TracerProvider.Shutdown 経由でエラーを発生させ、結合エラーの伝播を検証するために使う。
type errSpanExporter struct{ err error }

func (errSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (e errSpanExporter) Shutdown(context.Context) error { return e.err }

func TestNewProviderShutdowner(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("具象プロバイダを束ねた ProviderShutdowner を返す", func(t *testing.T) {
			t.Parallel()

			s := NewProviderShutdowner(sdktrace.NewTracerProvider(), sdkmetric.NewMeterProvider())

			require.NotNil(t, s)
		})
	})
}

func TestProviderShutdowner_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerProviderとMeterProviderをShutdownしエラーを返さない", func(t *testing.T) {
			t.Parallel()

			s := NewProviderShutdowner(sdktrace.NewTracerProvider(), sdkmetric.NewMeterProvider())

			require.NoError(t, s.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerProviderのShutdownが失敗した場合は結合エラーとして伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errors.New("span exporter shutdown failed")
			// バッチャ経由で失敗する exporter を仕込み、tp.Shutdown がエラーを返すようにする。
			tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(errSpanExporter{err: wantErr}))
			mp := sdkmetric.NewMeterProvider()

			// errors.Join は両引数（tp.Shutdown / mp.Shutdown）を評価してから結合するため、
			// tp が失敗しても mp の Shutdown は必ず呼ばれる（Go の引数評価保証）。
			err := NewProviderShutdowner(tp, mp).Shutdown(context.Background())

			require.ErrorContains(t, err, wantErr.Error())
		})
	})
}
