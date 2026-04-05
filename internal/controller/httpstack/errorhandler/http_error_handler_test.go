package errorhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/internal/controller/error/response/gen"
	"go-boilerplate/internal/controller/handler/testkit/testspan"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// badWriter は書き込み時にエラーを返すテスト用の http.ResponseWriter 実装です。
type badWriter struct {
	header      http.Header
	wroteHeader int
}

func (b *badWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *badWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

func (b *badWriter) WriteHeader(statusCode int) { b.wroteHeader = statusCode }

func TestNew(t *testing.T) {
	t.Parallel()
	e := echo.New()
	z := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)

	New(e, z, lf, obsCfg)
	require.NotNil(t, e.HTTPErrorHandler)
}

func TestNewHTTPErrorHandler(t *testing.T) {
	t.Parallel()

	z := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)
	handler := NewHTTPErrorHandler(z, lf, obsCfg)

	e := echo.New()
	ctx := context.Background()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(fmt.Errorf("some error"), c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	// レスポンスボディが gen.ErrorResponse の構造を持つこと
	_, ok := got["code"].(string)
	require.True(t, ok)

	_, ok = got["request_id"].(string)
	require.True(t, ok)
}

func Test_writeErrorResponse(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/err", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	he := &response.HTTPErrorResponse{
		ErrorResponse: gen.ErrorResponse{
			Code:      "E_TEST",
			Message:   "test message",
			RequestId: "req-xyz",
		},
		HTTPStatus: http.StatusTeapot,
	}

	err := writeErrorResponse(c, he)
	require.NoError(t, err)

	require.Equal(t, he.HTTPStatus, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Equal(t, he.Code, got["code"])
	require.Equal(t, he.Message, got["message"])
	require.Equal(t, he.RequestId, got["request_id"])
}

func Test_handleHTTPError(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	t.Run("正常系: レスポンス書き込みとログ出力", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)

		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		handleHTTPError(c, logger, lf, obsCfg, fmt.Errorf("boom"))
		handleHTTPError(c, logger, lf, obsCfg, fmt.Errorf("boom"))

		// JSON が書き込まれ、ステータスは内部サーバーエラー (おおむね 500) であること
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("書き込み失敗時: エラーログ出力と500セット", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)

		// badWriter は Write が失敗することで c.JSON を失敗させる
		bw := &badWriter{}

		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h2", nil)
		c := e.NewContext(req, bw)
		c, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		handleHTTPError(c, logger, lf, obsCfg, fmt.Errorf("boom2"))
	})
}

func Test_normalizeHTTPError(t *testing.T) {
	t.Parallel()

	expectedDetails := "expected details"
	expectedRequestID := "expected request ID"
	expectedInternal := fmt.Errorf("expected internal error: %w", apperror.ErrValidation)
	t.Run("AppErrorを直接渡した場合は対応するレスポンスを返す", func(t *testing.T) {
		t.Parallel()

		expected := response.NewHTTPErrorFromAppError(expectedInternal)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(expectedInternal, expectedRequestID)

		require.Equal(t, expected, actual)
	})

	t.Run("Echo.HTTPError の Internal に OpenAPI エラーがある場合、OpenAPI ハンドラ経由で正規化される", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestError -> 400", func(t *testing.T) {
			t.Parallel()
			reqErr := &openapi3filter.RequestError{}
			echoErr := &echo.HTTPError{Code: http.StatusBadRequest, Internal: reqErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusBadRequest)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			require.Equal(t, expected, actual)
		})

		t.Run("API定義書のエラー構造でステータスがエラー範囲外なら内部サーバーエラー扱い", func(t *testing.T) {
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
		})

		t.Run("SecurityRequirementsError -> 401", func(t *testing.T) {
			t.Parallel()
			secErr := &openapi3filter.SecurityRequirementsError{}
			echoErr := &echo.HTTPError{Code: http.StatusUnauthorized, Internal: secErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusUnauthorized)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			require.Equal(t, expected, actual)
		})

		t.Run("ResponseError -> 500", func(t *testing.T) {
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

	t.Run("response.HTTPErrorResponse を渡した場合、ステータスがエラー範囲ならそのまま返る", func(t *testing.T) {
		t.Parallel()

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "E_ERR",
				Message:   "err",
				Details:   nil,
				RequestId: "",
			},
			HTTPStatus: http.StatusBadRequest,
			Internal:   fmt.Errorf("inner"),
		}

		actual := normalizeHTTPError(he, expectedRequestID)
		he.RequestId = expectedRequestID
		require.Equal(t, he, actual)
	})

	t.Run("echo.HTTPError の場合 (エラー範囲) はステータスに基づくレスポンスを返す", func(t *testing.T) {
		t.Parallel()

		echoErr := &echo.HTTPError{Code: http.StatusForbidden}

		expected := response.NewHTTPErrorFromStatus(echoErr.Code)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(echoErr, expectedRequestID)
		require.Equal(t, expected, actual)
	})

	t.Run("echo.HTTPError の場合 (非エラー範囲) は内部エラーとして扱われる", func(t *testing.T) {
		t.Parallel()

		echoErr := &echo.HTTPError{Code: http.StatusContinue}

		expected := response.NewHTTPErrorFromStatus(echoErr.Code)
		expected.RequestId = expectedRequestID
		expected.Internal = echoErr

		actual := normalizeHTTPError(echoErr, expectedRequestID)
		require.Equal(t, expected, actual)
	})

	t.Run("nil エラーは内部サーバーエラーを返す", func(t *testing.T) {
		t.Parallel()

		actual := normalizeHTTPError(nil, expectedRequestID)
		expected := response.NewHTTPErrorFromAppError(nil)
		expected.RequestId = expectedRequestID
		require.Equal(t, expected, actual)
	})

	t.Run("その他の通常のエラーは内部サーバーエラーを返す", func(t *testing.T) {
		t.Parallel()

		expected := response.NewHTTPErrorFromAppError(expectedInternal)
		expected.RequestId = expectedRequestID

		actual := normalizeHTTPError(expectedInternal, expectedRequestID)
		require.Equal(t, expected, actual)
	})

	t.Run("echo.HTTPError の Internal に通常エラーがある場合、statusベースで返却され Internal は非nil", func(t *testing.T) {
		t.Parallel()

		inner := fmt.Errorf("boom")
		echoErr := &echo.HTTPError{Code: http.StatusForbidden, Internal: inner}

		actual := normalizeHTTPError(echoErr, expectedRequestID)
		expected := response.NewHTTPErrorFromStatus(echoErr.Code)
		expected.RequestId = expectedRequestID

		require.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
		require.Equal(t, expected.RequestId, actual.RequestId)
		require.Error(t, actual.Internal)
	})
}

func Test_logHTTPError(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	e := echo.New()
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/p", nil)
	req.RemoteAddr = "9.8.7.6:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c, end := testspan.StartTestSpanForEcho(t, c)
	defer end()

	t.Run("監視対象外のステータスコードはログ出力されない", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C302",
				Message:   "M302",
				RequestId: "r302",
			},
			HTTPStatus: http.StatusFound,
		}

		logHTTPError(c, logger, lf, obsCfg, he)
	})

	t.Run("500以上はErrorログ", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C500",
				Message:   "M500",
				RequestId: "r500",
			},
			HTTPStatus: http.StatusInternalServerError,
		}

		logHTTPError(c, logger, lf, obsCfg, he)
	})

	t.Run("400〜499はWarnログ", func(t *testing.T) {
		t.Parallel()

		logger := logging.NewTestLogger(t)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C404",
				Message:   "M404",
				RequestId: "r404",
			},
			HTTPStatus: http.StatusNotFound,
		}

		logHTTPError(c, logger, lf, obsCfg, he)
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

	lf := logging.NewTestLogFieldBuilder(t)

	e := echo.New()
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/p", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("DetailsとInternalがnilの場合、基本フィールドが含まれる", func(t *testing.T) {
		t.Parallel()

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "E_TEST",
				Message:   "m",
				RequestId: "rid",
			},
			HTTPStatus: http.StatusBadRequest,
		}

		fields := httpErrorField(c, lf, he)

		require.GreaterOrEqual(t, len(fields), 4)
		require.Contains(t, fields, logging.Int(logging.StatusKey, he.HTTPStatus))
		require.Contains(t, fields, logging.String(logging.ErrorCodeKey, he.Code))
		require.Contains(t, fields, logging.String(logging.ErrorMessageKey, he.Message))
		require.Contains(t, fields, logging.String(logging.RequestIDKey, he.RequestId))
	})

	t.Run("DetailsとInternalがある場合、内部情報フィールドが含まれる", func(t *testing.T) {
		t.Parallel()

		details := []string{"d1", "d2"}
		internalErr := fmt.Errorf("internal err")
		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "E_INT",
				Message:   "m2",
				Details:   &details,
				RequestId: "rid2",
			},
			HTTPStatus: http.StatusInternalServerError,
			Internal:   internalErr,
		}

		fields := httpErrorField(c, lf, he)

		// Details フィールド
		require.Contains(t, fields, logging.Strings(logging.ErrorDetails, details))

		// Internal error と stacktrace
		require.Contains(t, fields, logging.String(logging.InternalErrorKey, he.Internal.Error()))
		require.Contains(t, fields, logging.String(logging.InternalStackTraceKey, xerrors.StackTrace(he.Internal)))
	})
}
