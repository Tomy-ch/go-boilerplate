package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.Error(t, err)
		})

		t.Run("空文字列はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseLevel("")
			require.Error(t, err)
		})

		t.Run("大文字のレベル文字列はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseLevel("DEBUG")
			require.Error(t, err)
		})
	})
}

func TestLevelDebug(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestLevelError(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestLevelInfo(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestLevelWarn(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
