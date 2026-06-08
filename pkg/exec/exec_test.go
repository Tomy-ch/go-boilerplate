package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOS_Output(t *testing.T) {
	t.Parallel()

	t.Run("正常系_標準出力を返す", func(t *testing.T) {
		t.Parallel()
		out, err := OS{}.Output(context.Background(), "", "echo", []string{"hello"})
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(out))
	})

	t.Run("異常系_存在しないコマンドはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := OS{}.Output(context.Background(), "", "definitely-not-a-real-command-xyz", nil)
		require.Error(t, err)
	})

	t.Run("異常系_非ゼロ終了はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := OS{}.Output(context.Background(), "", "false", nil)
		require.Error(t, err)
	})
}
