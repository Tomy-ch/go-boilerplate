package observability

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/mock/gomock"
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

func Test_TracerProvider(t *testing.T) {
	// otel.SetTracerProvider / SetTextMapPropagator をグローバルに触るため Parallel 不可。

	t.Run("正常系", func(t *testing.T) {
		t.Run("Registrarにシャットダウンを登録し、伝播器を設定してグローバルなTracerProviderを返す", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			var shutdownFunc func(context.Context) error
			dummy := func(context.Context) error { return nil }
			mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				shutdownFunc = args[0].(func(context.Context) error)
			}).Times(1)

			tp, err := TracerProvider(mockReg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			_, ok := tp.(*sdktrace.TracerProvider)
			assert.True(t, ok)
			assert.Equal(t, tp, otel.GetTracerProvider())

			// W3C TraceContext + Baggage の伝播器が登録されていること。
			fields := otel.GetTextMapPropagator().Fields()
			assert.Contains(t, fields, "traceparent")
			assert.Contains(t, fields, "baggage")

			require.NotNil(t, shutdownFunc)
			require.NoError(t, shutdownFunc(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTEL_TRACES_EXPORTERが指定された場合はエラーを返す", func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", "invalid-exporter")

			ctrl := gomock.NewController(t)
			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)

			tp, err := TracerProvider(mockReg, newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, tp)
		})
	})
}

func Test_MeterProvider(t *testing.T) {
	// otel.SetMeterProvider をグローバルに触るため Parallel 不可。

	t.Run("正常系", func(t *testing.T) {
		t.Run("Registrarにシャットダウンを登録し、グローバルなMeterProviderを返す", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			var shutdownFunc func(context.Context) error
			dummy := func(context.Context) error { return nil }
			mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				shutdownFunc = args[0].(func(context.Context) error)
			}).Times(1)

			mp, err := MeterProvider(mockReg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			_, ok := mp.(*sdkmetric.MeterProvider)
			assert.True(t, ok)
			assert.Equal(t, mp, otel.GetMeterProvider())

			require.NotNil(t, shutdownFunc)
			require.NoError(t, shutdownFunc(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTEL_METRICS_EXPORTERが指定された場合はエラーを返す", func(t *testing.T) {
			t.Setenv("OTEL_METRICS_EXPORTER", "invalid-exporter")

			ctrl := gomock.NewController(t)
			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)

			mp, err := MeterProvider(mockReg, newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, mp)
		})
	})
}
