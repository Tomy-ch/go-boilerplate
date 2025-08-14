package response

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupErrorMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("既知のステータスコードの場合、対応するエラーを返す", func(t *testing.T) {
			t.Parallel()
			httpStatus := http.StatusForbidden

			expected := httpErrorMeta{
				Code:    codeAccessDenied,
				Message: errorMessageAccessDenied,
			}

			actual := lookupErrorMeta(httpStatus)

			require.Equal(t, expected, actual)
		})

		t.Run("未知のステータスコードの場合、内部サーバーエラーとして扱う", func(t *testing.T) {
			t.Parallel()
			httpStatus := 999

			expected := httpErrorMeta{
				Code:    codeInternalError,
				Message: errorMessageInternalError,
			}

			actual := lookupErrorMeta(httpStatus)

			require.Equal(t, expected, actual)
		})
	})
}
