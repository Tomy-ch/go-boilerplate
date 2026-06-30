package errorhandler

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/stretchr/testify/assert"
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

func (b *badWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func (b *badWriter) WriteHeader(statusCode int) { b.wroteHeader = statusCode }

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EchoのHTTPErrorHandlerが設定される", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			z := logging.NewTestLogger(t)
			obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
			lf := logging.NewTestLogFieldBuilder(t)

			New(e, z, lf, obsCfg)
			require.NotNil(t, e.HTTPErrorHandler)
		})
	})
}

func TestNewHTTPErrorHandler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一般エラーが500レスポンスとして書き出される", func(t *testing.T) {
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

			handler(errors.New("some error"), c)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

			code, ok := got["code"].(string)
			require.True(t, ok)
			// 一般エラーは内部エラーコードへ正規化される。
			assert.Equal(t, response.NewHTTPErrorFromAppError(nil).Code, code)

			requestID, ok := got["requestId"].(string)
			require.True(t, ok)
			// リクエストIDミドルウェア未経由のため空だが、文字列フィールドとして必ず出力される。
			assert.Empty(t, requestID)
		})
	})
}

func Test_writeErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HTTPErrorResponseがステータスとJSONボディとして書き出される", func(t *testing.T) {
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

			assert.Equal(t, he.HTTPStatus, rec.Code)

			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

			assert.Equal(t, he.Code, got["code"])
			assert.Equal(t, he.Message, got["message"])
			assert.Equal(t, he.RequestId, got["requestId"])
		})
	})
}

func Test_handleHTTPError(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("レスポンス書き込みとログ出力が行われる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			handleHTTPError(c, logger, lf, obsCfg, errors.New("boom"))

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		})

		t.Run("二重呼び出しでもレスポンスボディは1つだけ書かれる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			// 2 回目は ctxhelper.GetErrorHandledFromEcho ガードで抑止されるため、ボディは二重に書かれない。
			handleHTTPError(c, logger, lf, obsCfg, errors.New("boom"))
			handleHTTPError(c, logger, lf, obsCfg, errors.New("boom"))

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			dec := json.NewDecoder(rec.Body)
			var first map[string]any
			require.NoError(t, dec.Decode(&first))
			assert.False(t, dec.More())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き込み失敗時もエラーログ出力と500セットが行われる", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)

			bw := &badWriter{}

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h2", nil)
			c := e.NewContext(req, bw)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			handleHTTPError(c, logger, lf, obsCfg, errors.New("boom2"))

			assert.Equal(t, http.StatusInternalServerError, bw.wroteHeader)
		})
	})
}

func Test_normalizeHTTPError(t *testing.T) {
	t.Parallel()

	expectedDetails := "expected details"
	expectedRequestID := "expected request ID"
	expectedInternal := xerrors.Wrap(apperror.ErrValidation, "expected internal error")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("AppErrorを直接渡した場合は対応するレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			expected := response.NewHTTPErrorFromAppError(expectedInternal)
			expected.RequestId = expectedRequestID

			actual := normalizeHTTPError(expectedInternal, expectedRequestID)

			assert.Equal(t, http.StatusUnprocessableEntity, actual.HTTPStatus)
			assert.Equal(t, expected, actual)
		})

		t.Run("Echo.HTTPErrorのInternalにOpenAPIエラーがある場合_RequestErrorは400で正規化される", func(t *testing.T) {
			t.Parallel()
			reqErr := &openapi3filter.RequestError{}
			echoErr := &echo.HTTPError{Code: http.StatusBadRequest, Internal: reqErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)

			assert.Equal(t, http.StatusBadRequest, actual.HTTPStatus)
			expected := response.NewHTTPErrorFromStatus(http.StatusBadRequest, nil)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			assert.Equal(t, expected, actual)
		})

		t.Run("Echo.HTTPErrorのInternalにOpenAPIエラーがある場合_SecurityRequirementsErrorは401で正規化される", func(t *testing.T) {
			t.Parallel()
			secErr := &openapi3filter.SecurityRequirementsError{}
			echoErr := &echo.HTTPError{Code: http.StatusUnauthorized, Internal: secErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)

			assert.Equal(t, http.StatusUnauthorized, actual.HTTPStatus)
			expected := response.NewHTTPErrorFromStatus(http.StatusUnauthorized, nil)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			assert.Equal(t, expected, actual)
		})

		t.Run("Echo.HTTPErrorのInternalにOpenAPIエラーがある場合_ResponseErrorは500で正規化される", func(t *testing.T) {
			t.Parallel()
			respErr := &openapi3filter.ResponseError{}
			echoErr := &echo.HTTPError{Code: http.StatusInternalServerError, Internal: respErr}

			actual := normalizeHTTPError(echoErr, expectedRequestID)

			assert.Equal(t, http.StatusInternalServerError, actual.HTTPStatus)
			expected := response.NewHTTPErrorFromStatus(http.StatusInternalServerError, nil)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr
			assert.Equal(t, expected, actual)
		})

		t.Run("HTTPErrorResponseでステータスがエラー範囲外ならInternalを真として再正規化される", func(t *testing.T) {
			t.Parallel()

			expected := response.NewHTTPErrorFromAppError(
				expectedInternal,
				expectedDetails,
			)
			expected.RequestId = expectedRequestID

			unknownError := *expected
			unknownError.HTTPStatus = http.StatusContinue
			actual := normalizeHTTPError(&unknownError, expectedRequestID)

			assert.Equal(t, expected, actual)
		})

		t.Run("response.HTTPErrorResponseを渡した場合_ステータスがエラー範囲ならそのまま返る", func(t *testing.T) {
			t.Parallel()

			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "E_ERR",
					Message:   "err",
					Details:   nil,
					RequestId: "",
				},
				HTTPStatus: http.StatusBadRequest,
				Internal:   errors.New("inner"),
			}

			actual := normalizeHTTPError(he, expectedRequestID)

			// エラー範囲ステータスなら Code/Message/HTTPStatus は維持され、
			// RequestId は引数の値で確かに上書きされること（"" → expectedRequestID）を検証する。
			assert.Equal(t, expectedRequestID, actual.RequestId)
			assert.Equal(t, "E_ERR", actual.Code)
			assert.Equal(t, "err", actual.Message)
			assert.Equal(t, http.StatusBadRequest, actual.HTTPStatus)
		})

		t.Run("echo.HTTPErrorのエラー範囲はステータスに基づくレスポンスを返しInternalに文脈を保持する", func(t *testing.T) {
			t.Parallel()

			// Internal が nil の echo.HTTPError でも、文脈付き Internal が保持されること（回帰テスト）。
			echoErr := &echo.HTTPError{Code: http.StatusForbidden}

			expected := response.NewHTTPErrorFromStatus(echoErr.Code, nil)

			actual := normalizeHTTPError(echoErr, expectedRequestID)

			assert.Equal(t, http.StatusForbidden, actual.HTTPStatus)
			assert.Equal(t, expected.Code, actual.Code)
			assert.Equal(t, expected.Message, actual.Message)
			assert.Equal(t, expectedRequestID, actual.RequestId)
			require.Error(t, actual.Internal)
			assert.Contains(t, actual.Internal.Error(), "echo HTTP error")
		})

		t.Run("echo.HTTPErrorの非エラー範囲は内部エラーとして扱われる", func(t *testing.T) {
			t.Parallel()

			echoErr := &echo.HTTPError{Code: http.StatusContinue}

			expected := response.NewHTTPErrorFromStatus(echoErr.Code, nil)
			expected.RequestId = expectedRequestID
			expected.Internal = echoErr

			actual := normalizeHTTPError(echoErr, expectedRequestID)

			assert.Equal(t, http.StatusInternalServerError, actual.HTTPStatus)
			assert.Equal(t, expected, actual)
		})

		t.Run("nilエラーは内部サーバーエラーを返す", func(t *testing.T) {
			t.Parallel()

			actual := normalizeHTTPError(nil, expectedRequestID)
			expected := response.NewHTTPErrorFromAppError(nil)
			expected.RequestId = expectedRequestID

			assert.Equal(t, http.StatusInternalServerError, actual.HTTPStatus)
			assert.Equal(t, expected, actual)
		})

		t.Run("ドメイン語彙に該当しない素のエラーは内部サーバーエラー(500)を返す", func(t *testing.T) {
			t.Parallel()

			rawErr := errors.New("boom")

			actual := normalizeHTTPError(rawErr, expectedRequestID)

			assert.Equal(t, http.StatusInternalServerError, actual.HTTPStatus)
			assert.Equal(t, expectedRequestID, actual.RequestId)
		})

		t.Run("echo.HTTPErrorのInternalに通常エラーがある場合_statusベースで返却されInternalは非nilになる", func(t *testing.T) {
			t.Parallel()

			inner := errors.New("boom")
			echoErr := &echo.HTTPError{Code: http.StatusForbidden, Internal: inner}

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(echoErr.Code, nil)
			expected.RequestId = expectedRequestID

			assert.Equal(t, expected.HTTPStatus, actual.HTTPStatus)
			assert.Equal(t, expected.RequestId, actual.RequestId)
			require.Error(t, actual.Internal)
		})
	})
}

func Test_logHTTPError(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	newEchoCtx := func(t *testing.T) (echo.Context, func()) {
		t.Helper()
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/p", nil)
		req.RemoteAddr = "9.8.7.6:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		return testspan.StartTestSpanForEcho(t, c)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("監視対象外のステータスコードはログ出力されない", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			c, end := newEchoCtx(t)
			defer end()

			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "C302",
					Message:   "M302",
					RequestId: "r302",
				},
				HTTPStatus: http.StatusFound,
			}

			logHTTPError(c, logger, lf, obsCfg, he)

			assert.Equal(t, 0, observed.Len())
		})

		t.Run("500以上はErrorログとして出力される", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			c, end := newEchoCtx(t)
			defer end()

			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "C500",
					Message:   "M500",
					RequestId: "r500",
				},
				HTTPStatus: http.StatusInternalServerError,
			}

			logHTTPError(c, logger, lf, obsCfg, he)

			entries := observed.FilterMessage("errorhandler.server_error").All()
			require.Len(t, entries, 1)
			assert.Equal(t, 0, observed.FilterMessage("errorhandler.client_error").Len())
			assert.Equal(t, logging.EventTypeError, entries[0].ContextMap()[logging.EventTypeKey])
		})

		t.Run("400〜499はWarnログとして出力される", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			c, end := newEchoCtx(t)
			defer end()

			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "C404",
					Message:   "M404",
					RequestId: "r404",
				},
				HTTPStatus: http.StatusNotFound,
			}

			logHTTPError(c, logger, lf, obsCfg, he)

			entries := observed.FilterMessage("errorhandler.client_error").All()
			require.Len(t, entries, 1)
			assert.Equal(t, 0, observed.FilterMessage("errorhandler.server_error").Len())
			assert.Equal(t, logging.EventTypeError, entries[0].ContextMap()[logging.EventTypeKey])
		})
	})
}

func Test_isErrorStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("下限境界の400はtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isErrorStatus(400))
		})

		t.Run("上限境界の599はtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isErrorStatus(599))
		})

		t.Run("400未満の399はfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isErrorStatus(399))
		})

		t.Run("上限超過の600はfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isErrorStatus(600))
		})
	})
}

func Test_httpErrorField(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	newEchoCtx := func(t *testing.T) echo.Context {
		t.Helper()
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/p", nil)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DetailsとInternalがnilの場合、基本フィールドが含まれる", func(t *testing.T) {
			t.Parallel()

			c := newEchoCtx(t)
			he := &response.HTTPErrorResponse{
				ErrorResponse: gen.ErrorResponse{
					Code:      "E_TEST",
					Message:   "m",
					RequestId: "rid",
				},
				HTTPStatus: http.StatusBadRequest,
			}

			fields := httpErrorField(c, lf, he)

			assert.GreaterOrEqual(t, len(fields), 4)
			assert.Contains(t, fields, logging.Int(logging.StatusKey, he.HTTPStatus))
			assert.Contains(t, fields, logging.String(logging.ErrorCodeKey, he.Code))
			assert.Contains(t, fields, logging.String(logging.ErrorMessageKey, he.Message))
			assert.Contains(t, fields, logging.String(logging.RequestIDKey, he.RequestId))
		})

		t.Run("DetailsとInternalがある場合、内部情報フィールドが含まれる", func(t *testing.T) {
			t.Parallel()

			c := newEchoCtx(t)
			details := []string{"d1", "d2"}
			internalErr := errors.New("internal err")
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

			assert.Contains(t, fields, logging.Strings(logging.ErrorDetailsKey, details))
			assert.Contains(t, fields, logging.String(logging.InternalErrorKey, he.Internal.Error()))
			assert.Contains(t, fields, logging.Stacktrace(logging.InternalStackTraceKey, he.Internal))
		})
	})
}
