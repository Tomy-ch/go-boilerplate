package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのミドルウェアを返す", func(t *testing.T) {
			t.Parallel()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			assert.NotNil(t, Middleware(logger, lf))
		})
	})
}

func Test_loggingMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非運用系APIでは2xxをInfoレベルで出力する", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/mw", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := loggingMiddleware(logger, lf)(next)
			require.NoError(t, handler(c))

			handled := observed.FilterMessage("request handled")
			require.Equal(t, 1, handled.Len())
			assert.Equal(t, "info", handled.All()[0].Level.String())
		})

		t.Run("運用系APIではログが出力されない", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/health", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := loggingMiddleware(logger, lf)(next)
			require.NoError(t, handler(c))

			assert.Zero(t, observed.Len())
		})

		t.Run("5xxレスポンスではErrorレベルで出力される", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c echo.Context) error {
				return c.String(http.StatusInternalServerError, "err")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/mw", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := loggingMiddleware(logger, lf)(next)
			require.NoError(t, handler(c))

			handled := observed.FilterMessage("request handled")
			require.Equal(t, 1, handled.Len())
			assert.Equal(t, "error", handled.All()[0].Level.String())
		})
	})
}

func Test_requestLog_buildRequestLogFields(t *testing.T) {
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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("基本フィールドを返す（trace_id/span_idはLoggerがctxから注入する）", func(t *testing.T) {
			t.Parallel()

			l := requestLog{c: c, lf: lf}
			fields := l.buildRequestLogFields(time.Now())

			assert.Contains(t, fields, logging.String(logging.MethodKey, http.MethodGet))
		})
	})
}

func Test_requestLog_buildResponseLogFields(t *testing.T) {
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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("基本フィールドを返す（trace_id/span_idはLoggerがctxから注入する）", func(t *testing.T) {
			t.Parallel()

			c := newContext()
			expectedStatus := http.StatusCreated
			expectedRequestID := "req-123"

			c.Response().Status = expectedStatus
			c.Response().Header().Set("X-Request-Id", expectedRequestID)

			l := requestLog{c: c, lf: lf}
			fields := l.buildResponseLogFields(150 * time.Millisecond)

			assert.Contains(t, fields, logging.Int(logging.StatusKey, expectedStatus))
			assert.Contains(t, fields, logging.String(logging.RequestIDKey, expectedRequestID))
		})
	})
}
