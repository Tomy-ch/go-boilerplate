package logging

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	t.Run("production mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "production")

		logger, err := New(appCfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("development mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "development")

		logger, err := New(appCfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("unknown mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "unknown")

		logger, err := New(appCfg)
		require.Error(t, err)
		require.Nil(t, logger)
	})
}

func TestBuildLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系_設定が妥当な場合はLoggerを返す", func(t *testing.T) {
		t.Parallel()

		cfg := zap.Config{
			Level:         zap.NewAtomicLevelAt(zapcore.InfoLevel),
			Encoding:      "json",
			OutputPaths:   []string{"stdout"},
			EncoderConfig: zapcore.EncoderConfig{MessageKey: "msg"},
		}

		logger, err := buildLogger(cfg, zapcore.ErrorLevel)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("異常系_Build失敗時はLoggerを返さずエラーのみ返す", func(t *testing.T) {
		t.Parallel()

		// 未登録スキームの出力先で zap.Config.Build を失敗させる。
		// 修正前は中身 nil の Logger を返しており、初回ログ出力で panic していた。
		cfg := zap.Config{
			Level:         zap.NewAtomicLevelAt(zapcore.InfoLevel),
			Encoding:      "json",
			OutputPaths:   []string{"invalid-scheme://nowhere"},
			EncoderConfig: zapcore.EncoderConfig{MessageKey: "msg"},
		}

		logger, err := buildLogger(cfg, zapcore.ErrorLevel)
		require.Error(t, err)
		require.Nil(t, logger)
	})
}

func TestNewProductionLogger(t *testing.T) {
	logger, err := NewProductionLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestNewDevelopmentLogger(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
}
