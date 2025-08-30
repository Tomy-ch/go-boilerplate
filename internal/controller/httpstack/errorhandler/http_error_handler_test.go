package errorhandler

import (
	"fmt"
	"net/http"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/error/response"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()
	e := echo.New()
	z := zap.NewNop()
	New(e, z)
	require.NotNil(t, e.HTTPErrorHandler)
}

func Test_normalizeHTTPError(t *testing.T) {
	t.Parallel()

	expectedDetails := "expected details"
	expectedRequestID := "expected request ID"
	expectedInternal := fmt.Errorf("expected internal error: %w", apperror.ErrValidation)

	t.Run("API定義書で定義されたエラー構造でステータスがエラー範囲である場合、指定されたエラーが返る", func(t *testing.T) {
		t.Parallel()

		expected := response.New(
			expectedInternal,
			expectedDetails,
		)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(expected, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run(
		"API定義書で定義されたエラー構造でステータスがエラー範囲外の場合、内部サーバーエラーとして扱う",
		func(t *testing.T) {
			t.Parallel()

			expected := response.New(
				expectedInternal,
				expectedDetails,
			)
			expected.RequestId = expectedRequestID

			unknownError := *expected
			unknownError.HTTPStatus = http.StatusContinue
			actual := normalizeHTTPError(&unknownError, expectedRequestID)

			require.Equal(t, expected, actual)
		},
	)

	t.Run(
		"EchoのHTTPError構造でステータスがエラー範囲である場合、指定されたエラーがAPI定義書で定義されたエラー構造で返る",
		func(t *testing.T) {
			t.Parallel()

			echoError := &echo.HTTPError{
				Code: http.StatusForbidden,
			}

			expected := response.New(echoError)
			expected.RequestId = expectedRequestID

			actual := normalizeHTTPError(echoError, expectedRequestID)

			require.Equal(t, expected, actual)
		},
	)

	t.Run(
		"EchoのHTTPError構造でステータスがエラー範囲外の場合、内部サーバーエラーがAPI定義書で定義されたエラー構造で返る",
		func(t *testing.T) {
			t.Parallel()

			echoError := &echo.HTTPError{
				Code: http.StatusContinue,
			}

			expected := response.New(
				echoError,
			)
			expected.RequestId = expectedRequestID
			expected.Internal = echoError

			actual := normalizeHTTPError(echoError, expectedRequestID)

			require.Equal(t, expected, actual)
		},
	)

	t.Run("その他のエラーの場合、内部サーバーエラーがAPI定義書で定義されたエラー構造で返る", func(t *testing.T) {
		t.Parallel()

		expected := response.New(
			expectedInternal,
		)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(expectedInternal, expectedRequestID)

		require.Equal(t, expected, actual)
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
