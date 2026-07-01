package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOS_Output(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コマンドの標準出力を返す", func(t *testing.T) {
			t.Parallel()
			out, err := OS{}.Output(context.Background(), "", nil, "echo", []string{"hello"})
			require.NoError(t, err)
			assert.Equal(t, "hello\n", string(out))
		})

		t.Run("追加の環境変数がコマンドへ渡る", func(t *testing.T) {
			t.Parallel()
			out, err := OS{}.Output(context.Background(), "", []string{"FOO=bar"}, "sh", []string{"-c", `printf %s "$FOO"`})
			require.NoError(t, err)
			assert.Equal(t, "bar", string(out))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないコマンドはエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := OS{}.Output(context.Background(), "", nil, "definitely-not-a-real-command-xyz", nil)
			require.Error(t, err)
		})

		t.Run("非ゼロ終了コードはエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := OS{}.Output(context.Background(), "", nil, "false", nil)
			require.Error(t, err)
		})
	})
}
