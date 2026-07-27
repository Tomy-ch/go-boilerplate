package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v5"
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

		t.Run("非運用系APIでは2xxをInfoレベルで出力する", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c *echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/mw", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := Middleware(logger, lf)(next)
			require.NoError(t, handler(c))

			handled := observed.FilterMessage("request handled")
			require.Equal(t, 1, handled.Len())
			assert.Equal(t, "info", handled.All()[0].Level.String())
		})

		t.Run("運用系APIではログが出力されない", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c *echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/health", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := Middleware(logger, lf)(next)
			require.NoError(t, handler(c))

			assert.Zero(t, observed.Len())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("5xxレスポンスではErrorレベルで出力される", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)

			next := func(c *echo.Context) error {
				return c.String(http.StatusInternalServerError, "err")
			}

			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/mw", nil)
			req.RemoteAddr = "203.0.113.5:45678"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := Middleware(logger, lf)(next)
			require.NoError(t, handler(c))

			handled := observed.FilterMessage("request handled")
			require.Equal(t, 1, handled.Len())
			assert.Equal(t, "error", handled.All()[0].Level.String())
		})

		t.Run("レスポンスを取り出せない場合はログを出さず素通しする", func(t *testing.T) {
			t.Parallel()
			lf := logging.NewTestLogFieldBuilder(t)
			logger, observed := logging.NewObservedTestLogger(t)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw", nil)
			c := e.NewContext(req, httptest.NewRecorder())
			// Echo のレスポンスへ辿れないライタへ差し替える。
			c.SetResponse(httptest.NewRecorder())

			called := false
			handler := Middleware(logger, lf)(func(_ *echo.Context) error {
				called = true
				return nil
			})
			require.NoError(t, handler(c))

			assert.True(t, called, "ログを出せなくても後続ハンドラは実行される")
			assert.Equal(t, 0, observed.Len())
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
			assert.Contains(t, fields, logging.String(logging.URIKey, "/path?foo=bar"))
			assert.Contains(t, fields, logging.String(logging.RemoteIPKey, "1.2.3.4:5678"))
			assert.Contains(t, fields, logging.String(logging.HostKey, "example.local"))
			assert.Contains(t, fields, logging.String(logging.UserAgentKey, "ua-test"))
		})
	})
}

func Test_requestLog_buildResponseLogFields(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	// 各サブテストは並列実行されるため、共有の *echo.Context を使うと Response() の
	// ステータス/ヘッダ書き込みが競合する。サブテストごとに専用の Context を生成する。
	newContext := func() *echo.Context {
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
			expectedEventAt := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)

			res := server.ResponseOf(c)
			require.NotNil(t, res)
			res.Status = expectedStatus
			c.Response().Header().Set("X-Request-Id", expectedRequestID)

			l := requestLog{c: c, res: res, lf: lf}
			fields := l.buildResponseLogFields(expectedEventAt, 150*time.Millisecond)

			assert.Contains(t, fields, logging.Int(logging.StatusKey, expectedStatus))
			assert.Contains(t, fields, logging.String(logging.RequestIDKey, expectedRequestID))
			assert.Contains(t, fields, logging.Time(logging.EventAtKey, expectedEventAt))
		})
	})
}
