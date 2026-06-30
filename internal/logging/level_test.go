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

		t.Run("有効なレベル文字列を対応する Level に変換できる", func(t *testing.T) {
			t.Parallel()

			cases := map[string]Level{
				"debug": LevelDebug(),
				"info":  LevelInfo(),
				"warn":  LevelWarn(),
				"error": LevelError(),
			}
			for input, want := range cases {
				lv, err := ParseLevel(input)
				require.NoError(t, err)
				assert.Equal(t, want, lv)
			}
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
