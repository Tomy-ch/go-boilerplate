package redmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// observeCall は、Recorder.Observe の呼び出し引数を保持します。
type observeCall struct {
	method      string
	route       string
	statusCode  int
	statusClass string
	duration    time.Duration
}

// fakeRecorder は、Observe の呼び出しを記録するテスト用 Recorder です。
type fakeRecorder struct {
	mu    sync.Mutex
	calls []observeCall
}

// serveCfg は、ミドルウェア経由のリクエスト実行設定です。
type serveCfg struct {
	registerPath string
	requestPath  string
	handler      echo.HandlerFunc
}

func (f *fakeRecorder) Observe(method, route string, statusCode int, statusClass string, duration time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, observeCall{
		method:      method,
		route:       route,
		statusCode:  statusCode,
		statusClass: statusClass,
		duration:    duration,
	})
}

func (f *fakeRecorder) snapshot() []observeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]observeCall, len(f.calls))
	copy(out, f.calls)
	return out
}

// serve は、redmetrics ミドルウェアを適用した Echo に 1 リクエスト流し、記録された Observe 呼び出しを返します。
func serve(t *testing.T, cfg serveCfg) []observeCall {
	t.Helper()

	rec := &fakeRecorder{}
	e := echo.New()
	e.Use(Middleware(rec))
	if cfg.registerPath != "" {
		e.GET(cfg.registerPath, cfg.handler)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, cfg.requestPath, nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	return rec.snapshot()
}

// okHandler は、After フックを発火させるためボディ付きで応答するハンドラを返します。
func okHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("200応答でrequestが1件記録されstatus_classが2xxになる", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/users/:id",
				requestPath:  "/users/123",
				handler:      okHandler(),
			})

			require.Len(t, calls, 1)
			assert.Equal(t, http.MethodGet, calls[0].method)
			assert.Equal(t, "/users/:id", calls[0].route)
			assert.Equal(t, http.StatusOK, calls[0].statusCode)
			assert.Equal(t, "2xx", calls[0].statusClass)
			assert.GreaterOrEqual(t, calls[0].duration, time.Duration(0))
		})

		t.Run("routeにはpath_parameterの実値が入らずroute_patternが使われる", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/users/:id",
				requestPath:  "/users/123",
				handler:      okHandler(),
			})

			require.Len(t, calls, 1)
			assert.NotContains(t, calls[0].route, "123")
		})

		t.Run("query_stringはroute_labelに含まれない", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/users/:id",
				requestPath:  "/users/123?token=secret",
				handler:      okHandler(),
			})

			require.Len(t, calls, 1)
			assert.Equal(t, "/users/:id", calls[0].route)
			assert.NotContains(t, calls[0].route, "token")
			assert.NotContains(t, calls[0].route, "secret")
		})

		t.Run("複数回Writeしてもrequestは1件のみ記録される", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/stream",
				requestPath:  "/stream",
				handler: func(c echo.Context) error {
					c.Response().WriteHeader(http.StatusOK)
					for range 3 {
						if _, err := c.Response().Write([]byte("chunk")); err != nil {
							return err
						}
					}
					return nil
				},
			})

			require.Len(t, calls, 1)
			assert.Equal(t, http.StatusOK, calls[0].statusCode)
		})

		t.Run("204応答はAfterフックが発火せず計測されない", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/no-content",
				requestPath:  "/no-content",
				handler: func(c echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				},
			})

			assert.Empty(t, calls)
		})

		t.Run("route未一致の404ではrouteがunknownでstatus_classが4xxになる", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				requestPath: "/no-such-path",
			})

			require.Len(t, calls, 1)
			assert.Equal(t, routeUnknown, calls[0].route)
			assert.Equal(t, http.StatusNotFound, calls[0].statusCode)
			assert.Equal(t, "4xx", calls[0].statusClass)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("500応答でstatus_classが5xxになる", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/boom",
				requestPath:  "/boom",
				handler: func(_ echo.Context) error {
					return echo.NewHTTPError(http.StatusInternalServerError)
				},
			})

			require.Len(t, calls, 1)
			assert.Equal(t, http.StatusInternalServerError, calls[0].statusCode)
			assert.Equal(t, "5xx", calls[0].statusClass)
		})
	})

	t.Run("運用系パスは計測対象外", func(t *testing.T) {
		t.Parallel()

		t.Run("/metricsは計測されない", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/metrics",
				requestPath:  "/metrics",
				handler:      okHandler(),
			})

			assert.Empty(t, calls)
		})

		t.Run("/healthは計測されない", func(t *testing.T) {
			t.Parallel()

			calls := serve(t, serveCfg{
				registerPath: "/health",
				requestPath:  "/health",
				handler:      okHandler(),
			})

			assert.Empty(t, calls)
		})
	})
}
