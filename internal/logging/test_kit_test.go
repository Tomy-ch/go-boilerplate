package logging

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッドチェーンと出力でpanicしないLoggerを返す", func(t *testing.T) {
			t.Parallel()

			lg := NewTestLogger(t)

			require.NotPanics(t, func() {
				lg.Named("test").CallerSkip(1).Info(context.Background(), "message", String("key", "value"))
			})
		})
	})
}

func TestNewObservedTestLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("出力を捕捉するLoggerと観測ログを返す", func(t *testing.T) {
			t.Parallel()

			lg, observed := NewObservedTestLogger(t)
			require.NotNil(t, lg)
			require.NotNil(t, observed)

			lg.Info(context.Background(), "hello", String("key", "value"))

			logs := observed.All()
			require.Len(t, logs, 1)
			assert.Equal(t, "hello", logs[0].Message)
		})
	})
}

func TestNewObservedTestLoggerWithCaller(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("caller情報付きで出力を捕捉するLoggerと観測ログを返す", func(t *testing.T) {
			t.Parallel()

			lg, observed := NewObservedTestLoggerWithCaller(t)
			require.NotNil(t, lg)
			require.NotNil(t, observed)

			lg.Info(context.Background(), "hello", String("key", "value"))

			logs := observed.All()
			require.Len(t, logs, 1)
			assert.Equal(t, "hello", logs[0].Message)
			assert.True(t, logs[0].Caller.Defined)
		})
	})
}

func TestNewTestLogFieldBuilder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MockConfig由来のobsCfg/osCfgを保持するBuilderを返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			osCfg := config.NewOperatingSystemConfig(cfg)
			obsCfg := config.NewObservabilityConfig(cfg)

			actual := NewTestLogFieldBuilder(t)

			expected := &logFieldBuilder{
				obsCfg: obsCfg,
				osCfg:  osCfg,
			}
			assert.Equal(t, expected, actual)
		})
	})
}
