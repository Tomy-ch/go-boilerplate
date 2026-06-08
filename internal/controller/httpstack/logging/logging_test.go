package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-boilerplate/internal/controller/handler/testkit/testspan"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger(t)

	lf := logging.NewTestLogFieldBuilder(t)

	require.NotNil(t, Middleware(logger, lf))
}

func Test_loggingMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("非運用系APIではログが出力される", func(t *testing.T) {
		lf := logging.NewTestLogFieldBuilder(t)

		next := func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		}

		logger := logging.NewTestLogger(t)

		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/mw", nil)
		req.RemoteAddr = "203.0.113.5:45678"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := loggingMiddleware(logger, lf)(next)
		require.NoError(t, handler(c))
	})

	t.Run("運用系APIではログが出力されない", func(t *testing.T) {
		lf := logging.NewTestLogFieldBuilder(t)

		next := func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		}

		logger := logging.NewTestLogger(t)

		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/health", nil)
		req.RemoteAddr = "203.0.113.5:45678"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := loggingMiddleware(logger, lf)(next)
		require.NoError(t, handler(c))
	})
}

func Test_log_buildRequestLogFields(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	e := echo.New()
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path?foo=bar", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("User-Agent", "ua-test")
	req.Host = "example.local"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("スパンありはtrace/spanが含まれるリクエストログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		// リクエストコンテキストにテストスパンを埋め込む
		cWithSpan, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		tc := observability.ExtractTraceContext(cWithSpan.Request().Context())
		l := log{c: cWithSpan, lf: lf, traceCtx: tc}
		fields := l.buildRequestLogFields()
		require.NotEmpty(t, fields)
	})
}

func Test_log_buildResponseLogFields(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	// 各サブテストは並列実行されるため、共有の echo.Context を使うと Response() の
	// ステータス/ヘッダ書き込みが競合する。サブテストごとに専用の Context を生成する。
	newContext := func() echo.Context {
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/resp", nil)
		req.RemoteAddr = "5.6.7.8:9000"
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("スパンなしはtrace/spanは含まれないレスポンスログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		c := newContext()
		expectedStatus := http.StatusCreated
		expectedRequestID := "req-123"

		c.Response().Status = expectedStatus
		c.Response().Header().Set("X-Request-Id", expectedRequestID)

		l := log{c: c, lf: lf, traceCtx: observability.TraceContext{}}
		fields := l.buildResponseLogFields(150 * time.Millisecond)

		assert.Contains(t, fields, logging.Int(logging.StatusKey, expectedStatus))
		assert.Contains(t, fields, logging.String(logging.RequestIDKey, expectedRequestID))
	})

	t.Run("スパンありはtrace/spanが含まれるレスポンスログフィールドセットを返す", func(t *testing.T) {
		t.Parallel()

		c := newContext()
		expectedStatus := http.StatusAccepted
		expectedRequestID := "req-accepted"

		cWithSpan, end := testspan.StartTestSpanForEcho(t, c)
		defer end()

		cWithSpan.Response().Status = expectedStatus
		cWithSpan.Response().Header().Set("X-Request-Id", expectedRequestID)

		tc := observability.ExtractTraceContext(cWithSpan.Request().Context())
		l := log{c: cWithSpan, lf: lf, traceCtx: tc}
		fields := l.buildResponseLogFields(20 * time.Millisecond)

		assert.Contains(t, fields, logging.Int(logging.StatusKey, expectedStatus))
		assert.Contains(t, fields, logging.String(logging.RequestIDKey, expectedRequestID))
	})
}
