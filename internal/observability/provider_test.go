package observability

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func newTestResource(t *testing.T) *resource.Resource {
	t.Helper()

	appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
	res, err := NewResource(appCfg, system.NewBuildInfo())
	require.NoError(t, err)

	return res
}

func Test_NewResource(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アプリ設定とビルド情報からサービス識別属性を付与したリソースを生成する", func(t *testing.T) {
			t.Parallel()

			appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
			bi := system.NewBuildInfo()

			res, err := NewResource(appCfg, bi)

			require.NoError(t, err)
			require.NotNil(t, res)
			set := res.Set()

			name, ok := set.Value(semconv.ServiceNameKey)
			require.True(t, ok)
			assert.Equal(t, appCfg.Name(), name.AsString())

			env, ok := set.Value(semconv.DeploymentEnvironmentNameKey)
			require.True(t, ok)
			assert.Equal(t, appCfg.Env(), env.AsString())

			version, ok := set.Value(semconv.ServiceVersionKey)
			require.True(t, ok)
			assert.Equal(t, bi.Version(), version.AsString())

			revision, ok := set.Value(attribute.Key("service.revision"))
			require.True(t, ok)
			assert.Equal(t, bi.Revision(), revision.AsString())

			buildDate, ok := set.Value(attribute.Key("service.build_date"))
			require.True(t, ok)
			assert.Equal(t, bi.BuildDate(), buildDate.AsString())
		})
	})
}

func Test_noopSpanExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ExportSpansとShutdownは何もせずエラーを返さない", func(t *testing.T) {
			t.Parallel()

			exp := noopSpanExporter{}

			require.NoError(t, exp.ExportSpans(context.Background(), nil))
			require.NoError(t, exp.Shutdown(context.Background()))
		})
	})
}

//nolint:paralleltest // otel グローバル状態(TracerProvider/Propagator)を差し替えるため並列化不可
func Test_NewTracerProvider(t *testing.T) {
	// otel.SetTracerProvider / SetTextMapPropagator をグローバルに触るため Parallel 不可。

	t.Run("正常系", func(t *testing.T) {
		t.Run("伝播器を設定してグローバルなTracerProviderを構築し、Shutdown可能な具象を返す", func(t *testing.T) {
			prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
			t.Cleanup(func() {
				otel.SetTracerProvider(prevTP)
				otel.SetTextMapPropagator(prevProp)
			})

			tp, err := NewTracerProvider(newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			assert.Same(t, tp, otel.GetTracerProvider())

			// W3C TraceContext + Baggage の伝播器が登録されていること。
			fields := otel.GetTextMapPropagator().Fields()
			assert.Contains(t, fields, "traceparent")
			assert.Contains(t, fields, "baggage")

			// ライフサイクル登録は di 層に移譲したため、ここでは Shutdown が呼べることのみ確認する。
			require.NoError(t, tp.Shutdown(context.Background()))
		})

		t.Run("OTEL_TRACES_EXPORTERがnoneの場合もno-opとして構築しエラーを返さない", func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", "none")
			prevTP := otel.GetTracerProvider()
			t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

			tp, err := NewTracerProvider(newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			require.NoError(t, tp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTEL_TRACES_EXPORTERが指定された場合はエラーを返す", func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", "invalid-exporter")

			tp, err := NewTracerProvider(newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, tp)
		})
	})
}

func Test_ProvideTracerProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("具象TracerProviderをtrace.TracerProviderインターフェースとして返す", func(t *testing.T) {
			t.Parallel()

			tp := sdktrace.NewTracerProvider()

			got := ProvideTracerProvider(tp)

			assert.Same(t, tp, got)
		})
	})
}

//nolint:paralleltest // otel グローバル状態(MeterProvider)を差し替えるため並列化不可
func Test_NewMeterProvider(t *testing.T) {
	// otel.SetMeterProvider をグローバルに触るため Parallel 不可。

	t.Run("正常系", func(t *testing.T) {
		t.Run("グローバルなMeterProviderを構築し、Shutdown可能な具象を返す", func(t *testing.T) {
			prevMP := otel.GetMeterProvider()
			t.Cleanup(func() { otel.SetMeterProvider(prevMP) })

			mp, err := NewMeterProvider(newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			assert.Same(t, mp, otel.GetMeterProvider())

			// ライフサイクル登録は di 層に移譲したため、ここでは Shutdown が呼べることのみ確認する。
			require.NoError(t, mp.Shutdown(context.Background()))
		})

		t.Run("OTEL_METRICS_EXPORTERがnoneの場合もno-opとして構築しランタイム計装を行わない", func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXPORTER", "none")
			prevMP := otel.GetMeterProvider()
			t.Cleanup(func() { otel.SetMeterProvider(prevMP) })

			mp, err := NewMeterProvider(newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			require.NoError(t, mp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTEL_METRICS_EXPORTERが指定された場合はエラーを返す", func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXPORTER", "invalid-exporter")

			mp, err := NewMeterProvider(newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, mp)
		})
	})
}
