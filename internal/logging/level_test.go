package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("debugをLevelDebugに変換できる", func(t *testing.T) {
			t.Parallel()

			lv, err := ParseLevel("debug")
			require.NoError(t, err)
			assert.Equal(t, LevelDebug(), lv)
		})

		t.Run("infoをLevelInfoに変換できる", func(t *testing.T) {
			t.Parallel()

			lv, err := ParseLevel("info")
			require.NoError(t, err)
			assert.Equal(t, LevelInfo(), lv)
		})

		t.Run("warnをLevelWarnに変換できる", func(t *testing.T) {
			t.Parallel()

			lv, err := ParseLevel("warn")
			require.NoError(t, err)
			assert.Equal(t, LevelWarn(), lv)
		})

		t.Run("errorをLevelErrorに変換できる", func(t *testing.T) {
			t.Parallel()

			lv, err := ParseLevel("error")
			require.NoError(t, err)
			assert.Equal(t, LevelError(), lv)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈不能な文字列はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseLevel("invalid")
			require.ErrorIs(t, err, errUnsupportedLogLevel)
		})

		t.Run("空文字列はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseLevel("")
			require.ErrorIs(t, err, errUnsupportedLogLevel)
		})

		t.Run("大文字のレベル文字列はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseLevel("DEBUG")
			require.ErrorIs(t, err, errUnsupportedLogLevel)
		})
	})
}

func TestLevelDebug(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("zapcoreのDebugレベルを保持したLevelを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, zapcore.DebugLevel, LevelDebug().zl)
		})
	})
}

func TestLevelError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("zapcoreのErrorレベルを保持したLevelを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, zapcore.ErrorLevel, LevelError().zl)
		})
	})
}

func TestLevelInfo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("zapcoreのInfoレベルを保持したLevelを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, zapcore.InfoLevel, LevelInfo().zl)
		})

		t.Run("Levelのゼロ値と等価である", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, Level{}, LevelInfo())
		})
	})
}

func TestLevelWarn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("zapcoreのWarnレベルを保持したLevelを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, zapcore.WarnLevel, LevelWarn().zl)
		})
	})
}
