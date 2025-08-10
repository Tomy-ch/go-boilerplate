package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	errorresponse "boilerplate-go/internal/controller/handler/error/response"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_normalizeHTTPError(t *testing.T) {
	t.Parallel()

	expectedDetails := "expected details"
	expectedRequestID := "expected request ID"
	expectedInternal := errors.New("expected internal error")

	t.Run("API定義書で定義されたエラー構造でステータスがエラー範囲である場合、指定されたエラーが返る", func(t *testing.T) {
		t.Parallel()

		expected := errorresponse.New(http.StatusBadRequest, expectedInternal, expectedDetails)
		expected.RequestID = expectedRequestID

		actual := normalizeHTTPError(expected, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("API定義書で定義されたエラー構造でステータスがエラー範囲外の場合、内部サーバーエラーとして扱う", func(t *testing.T) {
		t.Parallel()

		expected := errorresponse.New(http.StatusInternalServerError, expectedInternal, expectedDetails)
		expected.RequestID = expectedRequestID

		unknownError := *expected
		unknownError.HTTPStatus = http.StatusContinue
		actual := normalizeHTTPError(&unknownError, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("EchoのHTTPError構造でステータスがエラー範囲である場合、指定されたエラーがAPI定義書で定義されたエラー構造で返る", func(t *testing.T) {
		t.Parallel()

		echoError := &echo.HTTPError{
			Code: http.StatusForbidden,
		}

		expected := errorresponse.New(http.StatusForbidden, echoError)
		expected.RequestID = expectedRequestID

		actual := normalizeHTTPError(echoError, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("EchoのHTTPError構造でステータスがエラー範囲外の場合、内部サーバーエラーがAPI定義書で定義されたエラー構造で返る", func(t *testing.T) {
		t.Parallel()

		echoError := &echo.HTTPError{
			Code: http.StatusContinue,
		}

		expected := errorresponse.New(http.StatusInternalServerError, echoError)
		expected.RequestID = expectedRequestID
		expected.Internal = echoError

		actual := normalizeHTTPError(echoError, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("その他のエラーの場合、内部サーバーエラーがAPI定義書で定義されたエラー構造で返る", func(t *testing.T) {
		t.Parallel()

		expected := errorresponse.New(http.StatusInternalServerError, expectedInternal)
		expected.RequestID = expectedRequestID

		actual := normalizeHTTPError(expectedInternal, expectedRequestID)

		require.Equal(t, expected, actual)
	})
}

func Test_getRequestID(t *testing.T) {
	t.Parallel()

	t.Run("X-Request-ID が設定されている場合", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderXRequestID, "test-request-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		id := getRequestID(c)
		require.Equal(t, "test-request-id", id)
	})

	t.Run("X-Request-ID が設定されていない場合", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		id := getRequestID(c)
		require.Equal(t, "", id)
	})
}

func Test_isErrorStatus(t *testing.T) {
	t.Parallel()

	t.Run("400〜599の範囲内のステータスコードはtrueを返す", func(t *testing.T) {
		t.Parallel()

		t.Run("400", func(t *testing.T) {
			t.Parallel()
			require.True(t, isErrorStatus(400))
		})
		t.Run("599", func(t *testing.T) {
			t.Parallel()
			require.True(t, isErrorStatus(599))
		})
	})

	t.Run("400未満および599を超えるステータスコードはfalseを返す", func(t *testing.T) {
		t.Parallel()

		t.Run("399", func(t *testing.T) {
			t.Parallel()
			require.False(t, isErrorStatus(399))
		})
		t.Run("600", func(t *testing.T) {
			t.Parallel()
			require.False(t, isErrorStatus(600))
		})
	})
}
