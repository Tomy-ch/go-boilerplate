package logging

import (
	"bytes"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

//nolint:paralleltest // config.SetApplicationMode でモック内部状態を書き換えるため並列化不可
func TestNew(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("本番モードの場合、Loggerを返す", func(t *testing.T) {
			appCfg := config.NewApplicationConfig(&config.Config{})
			appCfg.SetApplicationMode(t, "production")

			logger, err := New(appCfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})

		t.Run("開発モードの場合、Loggerを返す", func(t *testing.T) {
			appCfg := config.NewApplicationConfig(&config.Config{})
			appCfg.SetApplicationMode(t, "development")

			logger, err := New(appCfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("未知のモードの場合、エラーを返す", func(t *testing.T) {
			appCfg := config.NewApplicationConfig(&config.Config{})
			appCfg.SetApplicationMode(t, "unknown")

			logger, err := New(appCfg)
			require.Error(t, err)
			require.Nil(t, logger)
		})
	})
}

func TestBuildLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定レベル以上のログを出力先へ書き込む", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true)

			l.Info("hello")
			assert.Contains(t, buf.String(), "hello")
		})

		t.Run("指定レベル未満のログは書き込まれない", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true)

			l.Debug("should not appear")
			assert.Empty(t, buf.String())
		})
	})
}

func TestNewJSONLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON用Loggerを返す", func(t *testing.T) {
			t.Parallel()
			logger := NewJSONLogger(zapcore.InfoLevel, zapcore.ErrorLevel)
			require.NotNil(t, logger)
		})
	})
}

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("console用Loggerを返す", func(t *testing.T) {
			t.Parallel()
			logger := NewConsoleLogger(zapcore.DebugLevel, zapcore.WarnLevel)
			require.NotNil(t, logger)
		})
	})
}
