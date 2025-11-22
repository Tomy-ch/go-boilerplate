package errorhandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/error/response"
	"boilerplate-go/internal/controller/error/response/gen"

	"github.com/getkin/kin-openapi/openapi3filter"
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

func TestNewHTTPErrorHandler(t *testing.T) {
	t.Parallel()
	z := zap.NewNop()
	e := NewHTTPErrorHandler(z)
	require.NotNil(t, e)
}

func Test_normalizeHTTPError(t *testing.T) {
	t.Parallel()

	expectedDetails := "expected details"
	expectedRequestID := "expected request ID"
	expectedInternal := fmt.Errorf("expected internal error: %w", apperror.ErrValidation)

	t.Run("API定義書で定義されたエラー構造でステータスがエラー範囲である場合、指定されたエラーが返る", func(t *testing.T) {
		t.Parallel()

		expected := response.NewHTTPErrorFromAppError(
			expectedInternal,
		)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(expectedInternal, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("EchoのHTTPErrorが内部にOpenAPIエラーを持つ場合、normalizeOpenAPIError経由で正規化される", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestErrorの場合、BadRequestとして正規化される", func(t *testing.T) {
			t.Parallel()
			reqErr := &openapi3filter.RequestError{}
			echoErr := &echo.HTTPError{Code: http.StatusBadRequest, Internal: reqErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusBadRequest)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			require.Equal(t, expected, actual)
		})

		t.Run("SecurityRequirementsErrorの場合、Unauthorisedとして正規化される", func(t *testing.T) {
			t.Parallel()
			secErr := &openapi3filter.SecurityRequirementsError{}
			echoErr := &echo.HTTPError{Code: http.StatusUnauthorized, Internal: secErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusUnauthorized)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			require.Equal(t, expected, actual)
		})

		t.Run("ResponseErrorの場合、InternalServerErrorとして正規化される", func(t *testing.T) {
			t.Parallel()
			respErr := &openapi3filter.ResponseError{}
			echoErr := &echo.HTTPError{Code: http.StatusInternalServerError, Internal: respErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusInternalServerError)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			require.Equal(t, expected, actual)
		})
	})

	t.Run(
		"API定義書で定義されたエラー構造でステータスがエラー範囲外の場合、内部サーバーエラーとして扱う",
		func(t *testing.T) {
			t.Parallel()

			expected := response.NewHTTPErrorFromAppError(
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

			expected := response.NewHTTPErrorFromStatus(echoError.Code)
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

			expected := response.NewHTTPErrorFromStatus(echoError.Code)
			expected.RequestId = expectedRequestID
			expected.Internal = echoError

			actual := normalizeHTTPError(echoError, expectedRequestID)

			require.Equal(t, expected, actual)
		},
	)

	t.Run("その他のエラーの場合、内部サーバーエラーがAPI定義書で定義されたエラー構造で返る", func(t *testing.T) {
		t.Parallel()

		expected := response.NewHTTPErrorFromAppError(
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

func Test_httpErrorField(t *testing.T) {
	t.Parallel()

	e := echo.New()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DetailsとInternalがnilの場合、想定するフィールドが返る", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req.RemoteAddr = "1.2.3.4:1234"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "CODE",
					Message:   "MSG",
					Details:   nil,
					RequestId: "req-1",
				},
				HTTPStatus: http.StatusBadRequest,
				Internal:   nil,
			}

			actual := httpErrorField(c, he)

			expected := []zap.Field{
				zap.Int("status", he.HTTPStatus),
				zap.String("method", http.MethodGet),
				zap.String("path", "/foo"),
				zap.String("remote_ip", "1.2.3.4"),
				zap.String("request_id", he.RequestId),
				zap.String("error_code", he.Code),
				zap.String("error_message", he.Message),
			}

			require.Equal(t, expected, actual)
		})

		t.Run("DetailsとInternalがある場合、追加フィールドが含まれる", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/bar", nil)
			req.RemoteAddr = "5.6.7.8:4321"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			ds := []string{"d1", "d2"}
			internalErr := fmt.Errorf("inner: %w", apperror.ErrConflict)
			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "C2",
					Message:   "M2",
					Details:   &ds,
					RequestId: "req-2",
				},
				HTTPStatus: http.StatusInternalServerError,
				Internal:   internalErr,
			}

			actual := httpErrorField(c, he)

			expected := []zap.Field{
				zap.Int("status", he.HTTPStatus),
				zap.String("method", http.MethodPost),
				zap.String("path", "/bar"),
				zap.String("remote_ip", "5.6.7.8"),
				zap.String("request_id", he.RequestId),
				zap.String("error_code", he.Code),
				zap.String("error_message", he.Message),
				zap.Strings("error_details", ds),
				zap.String("internal_error", fmt.Sprintf("%v", he.Internal)),
			}

			require.Equal(t, expected, actual)
		})
	})
}
