package errorhandler

import (
	"context"
	"encoding/json"
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

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDetailPolicy は、details ゲートのテスト用に許可/拒否を固定する DetailPolicy スタブです。
type stubDetailPolicy struct{ allow bool }

// stubAllowPolicy は、Allow ヘッダーの解決結果を固定する AllowPolicy スタブです。
type stubAllowPolicy struct{ allow string }

// badWriter は書き込み時にエラーを返すテスト用の http.ResponseWriter 実装です。
type badWriter struct {
	header      http.Header
	wroteHeader int
}

func (s stubDetailPolicy) Allows(*http.Request) bool { return s.allow }

func (s stubAllowPolicy) Allow(*http.Request) string { return s.allow }

func (b *badWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *badWriter) Write([]byte) (int, error) { return 0, xerrors.New("write failed") }

func (b *badWriter) WriteHeader(statusCode int) { b.wroteHeader = statusCode }

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apperrorをHTTPステータスへ変換する本ハンドラが設定される", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			z := logging.NewTestLogger(t)
			obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
			lf := logging.NewTestLogFieldBuilder(t)

			New(e, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, z, lf, obsCfg)
			require.NotNil(t, e.HTTPErrorHandler)

			// echo 既定ハンドラは apperror を解釈しないため、ErrNotFound を 404 へ写像することで
			// 本ハンドラが設定されたことを挙動で確認する。
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/t", nil), rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()
			e.HTTPErrorHandler(c, apperror.ErrNotFound)
			assert.Equal(t, http.StatusNotFound, rec.Code)
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
			handler := NewHTTPErrorHandler(Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, z, lf, obsCfg)

			e := echo.New()
			ctx := context.Background()

			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/new", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler(c, xerrors.New("some error"))

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

func Test_responseCommitted(t *testing.T) {
	t.Parallel()

	newCtx := func() *echo.Context {
		e := echo.New()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/committed", nil)
		return e.NewContext(req, httptest.NewRecorder())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未送出のレスポンスはfalseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, responseCommitted(newCtx()))
		})

		t.Run("送出済みのレスポンスはtrueを返す", func(t *testing.T) {
			t.Parallel()

			c := newCtx()
			c.Response().WriteHeader(http.StatusOK)

			assert.True(t, responseCommitted(c))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Echoのレスポンスへ辿れない場合は未送出として扱う", func(t *testing.T) {
			t.Parallel()

			c := newCtx()
			c.SetResponse(httptest.NewRecorder())

			assert.False(t, responseCommitted(c))
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
					Code:      "E_TEST",
					Message:   "test message",
					RequestId: "req-xyz",
				},
				HTTPStatus: http.StatusTeapot,
			}

			err := writeErrorResponse(c, he, true)
			require.NoError(t, err)

			assert.Equal(t, he.HTTPStatus, rec.Code)

			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

			assert.Equal(t, he.Code, got["code"])
			assert.Equal(t, he.Message, got["message"])
			assert.Equal(t, he.RequestId, got["requestId"])
		})

		t.Run("exposeDetailsがtrueの場合、detailsがwireに含まれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/err", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			he := &response.HTTPErrorResponse{
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
					Code: "E_TEST", Message: "m", RequestId: "r", Details: new([]string{"firstName"}),
				},
				HTTPStatus: http.StatusUnprocessableEntity,
			}

			require.NoError(t, writeErrorResponse(c, he, true))

			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, []any{"firstName"}, got["details"])
			// resp 本体の details は温存され、ログ経路で使える
			require.NotNil(t, he.Details)
			assert.Equal(t, []string{"firstName"}, *he.Details)
		})

		t.Run("exposeDetailsがfalseの場合、wireからdetailsが落ちるがresp本体には残る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/err", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			he := &response.HTTPErrorResponse{
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
					Code: "E_TEST", Message: "m", RequestId: "r", Details: new([]string{"firstName"}),
				},
				HTTPStatus: http.StatusUnprocessableEntity,
			}

			require.NoError(t, writeErrorResponse(c, he, false))

			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			_, hasDetails := got["details"]
			assert.False(t, hasDetails)
			// wire だけ落とし、resp 本体(ログ用)は温存する
			require.NotNil(t, he.Details)
			assert.Equal(t, []string{"firstName"}, *he.Details)
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

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, xerrors.New("boom"))

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Equal(t, 1, observed.FilterMessage("errorhandler.server_error").Len())
		})

		t.Run("405はAllowPolicyが解決した許可メソッドをAllowヘッダーとして返す", func(t *testing.T) {
			t.Parallel()

			logger, _ := logging.NewObservedTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			handleHTTPError(
				c,
				Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{allow: "OPTIONS, GET"}},
				logger,
				lf,
				obsCfg,
				echo.ErrMethodNotAllowed,
			)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Equal(t, "OPTIONS, GET", rec.Header().Get(echo.HeaderAllow))
		})

		t.Run("レスポンス送出済みの405はAllowヘッダーもボディも書かない", func(t *testing.T) {
			t.Parallel()

			logger, _ := logging.NewObservedTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()
			c.Response().WriteHeader(http.StatusOK)

			handleHTTPError(
				c,
				Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{allow: "OPTIONS, GET"}},
				logger,
				lf,
				obsCfg,
				echo.ErrMethodNotAllowed,
			)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Header().Get(echo.HeaderAllow))
			assert.Empty(t, rec.Body.String())
		})

		t.Run("405以外はAllowPolicyが値を持っていてもAllowヘッダーを返さない", func(t *testing.T) {
			t.Parallel()

			logger, _ := logging.NewObservedTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			handleHTTPError(
				c,
				Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{allow: "OPTIONS, GET"}},
				logger,
				lf,
				obsCfg,
				echo.ErrNotFound,
			)

			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Empty(t, rec.Header().Get(echo.HeaderAllow))
		})

		t.Run("details付きエラーはpolicyが拒否するとwireから落ちるがログには残る", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			metaErr := apperror.WithDetails(xerrors.Wrap(apperror.ErrValidation, "invalid"), "firstName")
			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: false}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, metaErr)

			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			_, hasDetails := got["details"]
			assert.False(t, hasDetails)

			entries := observed.FilterMessage("errorhandler.client_error").All()
			require.Len(t, entries, 1)
			assert.Contains(t, entries[0].ContextMap()[logging.ErrorDetailsKey], "firstName")
		})

		t.Run("details付きエラーはpolicyが許可するとwireにdetailsが載る", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/h", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c, end := testspan.StartTestSpanForEcho(t, c)
			defer end()

			metaErr := apperror.WithDetails(xerrors.Wrap(apperror.ErrValidation, "invalid"), "firstName")
			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, metaErr)

			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			var got map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, []any{"firstName"}, got["details"])
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
			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, xerrors.New("boom"))
			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, xerrors.New("boom"))

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

			handleHTTPError(c, Policies{Detail: stubDetailPolicy{allow: true}, Allow: stubAllowPolicy{}}, logger, lf, obsCfg, xerrors.New("boom2"))

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

		t.Run("Meta付きAppErrorを渡した場合はDetailsが反映される", func(t *testing.T) {
			t.Parallel()

			joined := xerrors.Join(
				xerrors.Wrap(apperror.ErrValidation, "first name failed"),
				xerrors.Wrap(apperror.ErrValidation, "email failed"),
			)
			metaErr := apperror.WithDetails(joined, "firstName", "email")

			expected := response.NewHTTPErrorFromAppError(metaErr)
			expected.RequestId = expectedRequestID

			actual := normalizeHTTPError(metaErr, expectedRequestID)

			assert.Equal(t, http.StatusUnprocessableEntity, actual.HTTPStatus)
			require.NotNil(t, actual.Details)
			assert.Equal(t, []string{"firstName", "email"}, *actual.Details)
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
					Code:      "E_ERR",
					Message:   "err",
					Details:   nil,
					RequestId: "",
				},
				HTTPStatus: http.StatusBadRequest,
				Internal:   xerrors.New("inner"),
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

			rawErr := xerrors.New("boom")

			actual := normalizeHTTPError(rawErr, expectedRequestID)

			assert.Equal(t, http.StatusInternalServerError, actual.HTTPStatus)
			assert.Equal(t, expectedRequestID, actual.RequestId)
		})

		t.Run("echo.HTTPErrorが通常エラーを内包する場合_statusベースで返却されInternalは非nilになる", func(t *testing.T) {
			t.Parallel()

			inner := xerrors.New("boom")
			echoErr := echo.NewHTTPError(http.StatusForbidden, "").Wrap(inner)

			actual := normalizeHTTPError(echoErr, expectedRequestID)
			expected := response.NewHTTPErrorFromStatus(http.StatusForbidden, nil)
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

	newEchoCtx := func(t *testing.T) (*echo.Context, func()) {
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
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

	newEchoCtx := func(t *testing.T) *echo.Context {
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
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
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
			internalErr := xerrors.New("internal err")
			he := &response.HTTPErrorResponse{
				ErrorResponseWithDetails: gen.ErrorResponseWithDetails{
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
