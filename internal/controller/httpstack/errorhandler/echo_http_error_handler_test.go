package errorhandler

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_normalizeEchoHTTPError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EchoHTTPErrorでステータス範囲内の場合、NewHTTPErrorFromStatusベースのレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			inner := xerrors.New("inner failure")
			ehe := echo.NewHTTPError(http.StatusForbidden, "").Wrap(inner)

			actual := normalizeEchoHTTPError(ehe)
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusForbidden, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.Message, actual.Message)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
			assert.Nil(t, actual.Details)
			require.Error(t, actual.Internal)
			assert.Contains(t, actual.Internal.Error(), "inner failure")
			assert.Contains(t, actual.Internal.Error(), "echo HTTP error")
		})

		t.Run("detailsを渡した場合、Detailsにセットされる", func(t *testing.T) {
			t.Parallel()

			inner := xerrors.New("inner2")
			ehe := echo.NewHTTPError(http.StatusConflict, "").Wrap(inner)

			actual := normalizeEchoHTTPError(ehe, "d1", "d2")
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusConflict, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.Message, actual.Message)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
			require.NotNil(t, actual.Details)
			assert.Equal(t, []string{"d1", "d2"}, *actual.Details)
			require.Error(t, actual.Internal)
			assert.Contains(t, actual.Internal.Error(), "inner2")
		})

		t.Run("Echoの定義済みエラーの場合、ステータスが解決されレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			actual := normalizeEchoHTTPError(echo.ErrNotFound)
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusNotFound, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
		})

		t.Run("メソッド不許可の場合、405とMETHOD_NOT_ALLOWEDが返る", func(t *testing.T) {
			t.Parallel()

			actual := normalizeEchoHTTPError(echo.ErrMethodNotAllowed)
			require.NotNil(t, actual)

			assert.Equal(t, http.StatusMethodNotAllowed, actual.HTTPStatus)
			assert.Equal(t, "METHOD_NOT_ALLOWED", actual.Code)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非EchoHTTPErrorの場合、nilが返る", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, normalizeEchoHTTPError(xerrors.New("just an error")))
		})

		t.Run("EchoHTTPErrorだがステータス範囲外の場合、nilが返る", func(t *testing.T) {
			t.Parallel()
			ehe := echo.NewHTTPError(http.StatusContinue, "")
			assert.Nil(t, normalizeEchoHTTPError(ehe))
		})
	})
}
