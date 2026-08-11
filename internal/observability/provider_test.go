package observability

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

// newTestObsCfg は、mock 設定から ObservabilityConfig を返す（既定: trace/metric ともに otlp 有効）。
func newTestObsCfg(t *testing.T) *config.ObservabilityConfig {
	t.Helper()
	return config.NewObservabilityConfig(config.MockConfigForTest(t))
}

// shutdownMeterProvider は、PeriodicReader の最終 export を伴う Shutdown を
// ローカル境界の短い deadline で打ち切る（送出可否は検証対象外。goroutine の後始末のみが目的）。
func shutdownMeterProvider(t *testing.T, mp *sdkmetric.MeterProvider) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = mp.Shutdown(ctx)
}

func TestNewResource(t *testing.T) {
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

func TestNewTextMapPropagator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TraceContextとBaggageの複合伝播器を構築する", func(t *testing.T) {
			t.Parallel()

			prop := NewTextMapPropagator()
			require.NotNil(t, prop)

			fields := prop.Fields()
			assert.Contains(t, fields, "traceparent")
			assert.Contains(t, fields, "baggage")
		})
	})
}

//nolint:paralleltest // otel グローバル状態(TracerProvider/Propagator)を差し替えるため並列化不可
func TestNewTracerProvider(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("trace有効(http)で伝播器を設定しグローバルなTracerProviderを構築する", func(t *testing.T) {
			prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
			t.Cleanup(func() {
				otel.SetTracerProvider(prevTP)
				otel.SetTextMapPropagator(prevProp)
			})

			tp, err := NewTracerProvider(newTestObsCfg(t), newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			assert.Same(t, tp, otel.GetTracerProvider())

			fields := otel.GetTextMapPropagator().Fields()
			assert.Contains(t, fields, "traceparent")
			assert.Contains(t, fields, "baggage")

			require.NoError(t, tp.Shutdown(context.Background()))
		})

		t.Run("trace有効(grpc)でも構築できる", func(t *testing.T) {
			prevTP := otel.GetTracerProvider()
			t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)

			tp, err := NewTracerProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			require.NoError(t, tp.Shutdown(context.Background()))
		})

		t.Run("trace無効ならBatcherを付けずに構築する", func(t *testing.T) {
			prevTP := otel.GetTracerProvider()
			t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityTracesExporter(t, "")

			tp, err := NewTracerProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, tp)
			require.NoError(t, tp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTLPプロトコルが指定された場合はエラーを返す", func(t *testing.T) {
			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			tp, err := NewTracerProvider(obsCfg, newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, tp)
		})
	})
}

func Test_newSpanExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http/protobuf 指定で HTTP exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolHTTP)

			exp, err := newSpanExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})

		t.Run("grpc 指定で gRPC exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)

			exp, err := newSpanExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のプロトコル指定はエラーを返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			exp, err := newSpanExporter(context.Background(), obsCfg)

			require.ErrorIs(t, err, errInvalidOTLPProtocol)
			assert.Nil(t, exp)
		})
	})
}

func Test_newMetricExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http/protobuf 指定で HTTP exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolHTTP)

			exp, err := newMetricExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})

		t.Run("grpc 指定で gRPC exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)

			exp, err := newMetricExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のプロトコル指定はエラーを返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			exp, err := newMetricExporter(context.Background(), obsCfg)

			require.ErrorIs(t, err, errInvalidOTLPProtocol)
			assert.Nil(t, exp)
		})
	})
}

func Test_newMetricReader(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("exporter 構築成功なら PeriodicReader を返す", func(t *testing.T) {
			t.Parallel()

			reader, err := newMetricReader(context.Background(), newTestObsCfg(t))

			require.NoError(t, err)
			require.NotNil(t, reader)
			require.NoError(t, reader.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("exporter 構築失敗のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			reader, err := newMetricReader(context.Background(), obsCfg)

			require.ErrorIs(t, err, errInvalidOTLPProtocol)
			assert.Nil(t, reader)
		})
	})
}

func TestProvideTracerProvider(t *testing.T) {
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

func TestProvideMeterProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("具象MeterProviderをmetric.MeterProviderインターフェースとして返す", func(t *testing.T) {
			t.Parallel()

			mp := sdkmetric.NewMeterProvider()

			got := ProvideMeterProvider(mp)

			assert.Same(t, mp, got)
		})
	})
}

//nolint:paralleltest // otel グローバル状態(MeterProvider)を差し替えるため並列化不可
func TestNewMeterProvider(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("metric有効(http)でグローバルなMeterProviderを構築する", func(t *testing.T) {
			prevMP := otel.GetMeterProvider()
			t.Cleanup(func() { otel.SetMeterProvider(prevMP) })

			mp, err := NewMeterProvider(newTestObsCfg(t), newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			assert.Same(t, mp, otel.GetMeterProvider())

			shutdownMeterProvider(t, mp)
		})

		t.Run("metric有効(grpc)でも構築できる", func(t *testing.T) {
			prevMP := otel.GetMeterProvider()
			t.Cleanup(func() { otel.SetMeterProvider(prevMP) })

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)

			mp, err := NewMeterProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			shutdownMeterProvider(t, mp)
		})

		t.Run("metric無効ならReaderを付けずランタイム計装も行わない", func(t *testing.T) {
			prevMP := otel.GetMeterProvider()
			t.Cleanup(func() { otel.SetMeterProvider(prevMP) })

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityMetricsExporter(t, "")

			mp, err := NewMeterProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, mp)
			require.NoError(t, mp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("不正なOTLPプロトコルが指定された場合はエラーを返す", func(t *testing.T) {
			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			mp, err := NewMeterProvider(obsCfg, newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, mp)
		})
	})
}
