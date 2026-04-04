package errorhandler

import (
	"fmt"
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/error/response"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_normalizeEchoHTTPError_Specific(t *testing.T) {
	t.Parallel()

	t.Run("非EchoHTTPErrorの場合、nilが返る", func(t *testing.T) {
		t.Parallel()

		actual := normalizeEchoHTTPError(fmt.Errorf("just an error"))
		require.Nil(t, actual)
	})

	t.Run("EchoHTTPErrorだがステータス範囲外の場合、nilが返る", func(t *testing.T) {
		t.Parallel()

		ehe := &echo.HTTPError{Code: http.StatusContinue}
		actual := normalizeEchoHTTPError(ehe)
		require.Nil(t, actual)
	})

	t.Run("正常系: EchoHTTPErrorでステータス範囲内の場合、NewHTTPErrorFromStatusベースのレスポンスが返る", func(t *testing.T) {
		t.Parallel()

		inner := fmt.Errorf("inner failure")
		ehe := &echo.HTTPError{Code: http.StatusForbidden, Internal: inner}

		actual := normalizeEchoHTTPError(ehe)
		require.NotNil(t, actual)

		expectedBase := response.NewHTTPErrorFromStatus(http.StatusForbidden)
		require.Equal(t, expectedBase.Code, actual.Code)
		require.Equal(t, expectedBase.Message, actual.Message)
		require.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
		require.Nil(t, actual.Details)
		require.Error(t, actual.Internal)
		require.Contains(t, actual.Internal.Error(), "inner failure")
		require.Contains(t, actual.Internal.Error(), "echo HTTP error")
	})

	t.Run("正常系: detailsを渡した場合、Detailsにセットされる", func(t *testing.T) {
		t.Parallel()

		inner := fmt.Errorf("inner2")
		ehe := &echo.HTTPError{Code: http.StatusConflict, Internal: inner}

		actual := normalizeEchoHTTPError(ehe, "d1", "d2")
		require.NotNil(t, actual)

		expectedBase := response.NewHTTPErrorFromStatus(http.StatusConflict)
		require.Equal(t, expectedBase.Code, actual.Code)
		require.Equal(t, expectedBase.Message, actual.Message)
		require.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
		require.NotNil(t, actual.Details)
		require.Equal(t, []string{"d1", "d2"}, *actual.Details)
		require.Error(t, actual.Internal)
		require.Contains(t, actual.Internal.Error(), "inner2")
	})
}
