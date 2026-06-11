package observability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getCallerFullName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("呼び出し元の完全修飾関数名を非空文字列で返す", func(t *testing.T) {
			t.Parallel()
			got := getCallerFullName()
			require.NotEmpty(t, got)
		})
	})
}
