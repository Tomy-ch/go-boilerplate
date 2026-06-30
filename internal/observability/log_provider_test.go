package observability

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// shutdownLoggerProvider は、BatchProcessor の最終 export を伴う Shutdown を
// ローカル境界の短い deadline で打ち切る（送出可否は検証対象外。goroutine の後始末のみが目的）。
func shutdownLoggerProvider(t *testing.T, lp *sdklog.LoggerProvider) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = lp.Shutdown(ctx)
}

func newTestAppCfg(t *testing.T) *config.ApplicationConfig {
	t.Helper()
	return config.NewApplicationConfig(config.MockConfigForTest(t))
}

func Test_NewLoggerProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("log有効(http)でLoggerProviderを構築する", func(t *testing.T) {
			t.Parallel()

			lp, err := NewLoggerProvider(newTestObsCfg(t), newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, lp)
			shutdownLoggerProvider(t, lp)
		})

		t.Run("log有効(grpc)でも構築できる", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)

			lp, err := NewLoggerProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, lp)
			shutdownLoggerProvider(t, lp)
		})

		t.Run("log無効ならProcessorを付けずに構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityLogsExporter(t, "")

			lp, err := NewLoggerProvider(obsCfg, newTestResource(t))

			require.NoError(t, err)
			require.NotNil(t, lp)
			require.NoError(t, lp.Shutdown(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なOTLPプロトコルが指定された場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, "invalid-protocol")

			lp, err := NewLoggerProvider(obsCfg, newTestResource(t))

			require.Error(t, err)
			assert.Nil(t, lp)
		})
	})
}

func Test_newLogExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("endpoint未設定(http)でも既定値で exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolHTTP)
			obsCfg.SetObservabilityOTLPEndpoint(t, "")

			exp, err := newLogExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})

		t.Run("endpoint未設定(grpc)でも既定値で exporter を構築する", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityOTLPProtocol(t, protocolGRPC)
			obsCfg.SetObservabilityOTLPEndpoint(t, "")

			exp, err := newLogExporter(context.Background(), obsCfg)

			require.NoError(t, err)
			require.NotNil(t, exp)
			require.NoError(t, exp.Shutdown(context.Background()))
		})
	})
}

func Test_ensureOTLPPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"正常系_path無しなら補う", "http://observability:4318", "http://observability:4318/v1/logs"},
		{"正常系_末尾スラッシュも補う", "http://observability:4318/", "http://observability:4318/v1/logs"},
		{"正常系_path有りならそのまま", "http://observability:4318/custom", "http://observability:4318/custom"},
		{"正常系_パース不能ならそのまま", "://bad", "://bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ensureOTLPPath(tc.in, otlpLogsPath))
		})
	}
}

func Test_NewLogCore(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("log有効なら otelzap core を返す", func(t *testing.T) {
			t.Parallel()

			lp := sdklog.NewLoggerProvider()
			core := NewLogCore(newTestObsCfg(t), newTestAppCfg(t), lp)

			assert.NotNil(t, core)
		})

		t.Run("log無効なら nil を返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := newTestObsCfg(t)
			obsCfg.SetObservabilityLogsExporter(t, "")
			lp := sdklog.NewLoggerProvider()

			core := NewLogCore(obsCfg, newTestAppCfg(t), lp)

			assert.Nil(t, core)
		})
	})
}
