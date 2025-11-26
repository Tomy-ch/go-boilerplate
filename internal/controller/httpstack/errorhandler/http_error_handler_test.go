package errorhandler

import (
	"bytes"
	"encoding/json"
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
	"go.uber.org/zap/zapcore"
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
	z := zap.NewNop()
	New(e, z)
	require.NotNil(t, e.HTTPErrorHandler)
}

func TestNewHTTPErrorHandler(t *testing.T) {
	t.Parallel()

	z := zap.NewNop()
	handler := NewHTTPErrorHandler(z)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/new", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler(fmt.Errorf("some error"), c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	// レスポンスボディが gen.ErrorResponse の構造を持つこと
	_, ok := got["code"].(string)
	require.True(t, ok)

	_, ok2 := got["request_id"].(string)
	require.True(t, ok2)
}

func Test_writeErrorResponse(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)
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

	newTestLogger := func(buf *bytes.Buffer) *zap.Logger {
		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = ""
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(buf),
			zapcore.DebugLevel,
		)
		return zap.New(core)
	}

	t.Run("正常系: レスポンス書き込みとログ出力", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := newTestLogger(&buf)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/h", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handleHTTPError(logger, c, fmt.Errorf("boom"))

		// JSON が書き込まれ、ステータスは内部サーバーエラー (おおむね 500) であること
		require.Equal(t, http.StatusInternalServerError, rec.Code)

		out := buf.String()
		require.Contains(t, out, `"msg":"http_error"`)
		require.Contains(t, out, `"level":"error"`)
	})

	t.Run("書き込み失敗時: エラーログ出力と500セット", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := newTestLogger(&buf)

		// badWriter は Write が失敗することで c.JSON を失敗させる
		bw := &badWriter{}

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/h2", nil)
		c := e.NewContext(req, bw)

		handleHTTPError(logger, c, fmt.Errorf("boom2"))

		// writeErrorResponse が失敗した場合、handleHTTPError は Error ログを出す
		out := buf.String()
		require.Contains(t, out, "failed to write error response")
		require.Equal(t, http.StatusInternalServerError, bw.wroteHeader)
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

	newTestLogger := func(buf *bytes.Buffer) *zap.Logger {
		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = ""
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(buf),
			zapcore.DebugLevel,
		)
		return zap.New(core)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.RemoteAddr = "9.8.7.6:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("500以上はErrorログ", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := newTestLogger(&buf)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C500",
				Message:   "M500",
				RequestId: "r500",
			},
			HTTPStatus: http.StatusInternalServerError,
		}

		logHTTPError(logger, c, he)

		out := buf.String()
		require.Contains(t, out, `"msg":"http_error"`)
		require.Contains(t, out, `"level":"error"`)
		require.Contains(t, out, `"status":`+fmt.Sprint(he.HTTPStatus))
	})

	t.Run("400〜499はWarnログ", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := newTestLogger(&buf)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C404",
				Message:   "M404",
				RequestId: "r404",
			},
			HTTPStatus: http.StatusNotFound,
		}

		logHTTPError(logger, c, he)

		out := buf.String()
		require.Contains(t, out, `"msg":"http_error"`)
		require.Contains(t, out, `"level":"warn"`)
		require.Contains(t, out, `"status":`+fmt.Sprint(he.HTTPStatus))
	})

	t.Run("それ以外はInfoログ", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := newTestLogger(&buf)

		he := &response.HTTPErrorResponse{
			ErrorResponse: gen.ErrorResponse{
				Code:      "C200",
				Message:   "M200",
				RequestId: "r200",
			},
			HTTPStatus: http.StatusOK,
		}

		logHTTPError(logger, c, he)

		out := buf.String()
		require.Contains(t, out, `"msg":"http_error"`)
		require.Contains(t, out, `"level":"info"`)
		require.Contains(t, out, `"status":`+fmt.Sprint(he.HTTPStatus))
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
