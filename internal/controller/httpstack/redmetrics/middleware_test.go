package redmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_redmetrics "go-boilerplate/internal/controller/httpstack/redmetrics/mock"
)

// serveCfg は、ミドルウェア経由のリクエスト実行設定です。
type serveCfg struct {
	registerPath string
	requestPath  string
	handler      echo.HandlerFunc
}

// serve は、redmetrics ミドルウェアを適用した Echo に rec を計測先として 1 リクエスト流します。
// 計測の検証は rec（生成 mock）に設定した EXPECT で行います。
func serve(t *testing.T, rec Recorder, cfg serveCfg) {
	t.Helper()

	e := echo.New()
	e.Use(Middleware(rec))
	if cfg.registerPath != "" {
		e.GET(cfg.registerPath, cfg.handler)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, cfg.requestPath, nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
}

// okHandler は、After フックを発火させるためボディ付きで応答するハンドラを返します。
func okHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("200応答でrequestが1件記録されstatus_classが2xxになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			rec.EXPECT().Observe(http.MethodGet, "/users/:id", http.StatusOK, "2xx", gomock.Any()).Times(1)

			serve(t, rec, serveCfg{
				registerPath: "/users/:id",
				requestPath:  "/users/123",
				handler:      okHandler(),
			})
		})

		t.Run("query_stringはroute_labelに含まれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// query string（token=secret）は route label に混入しない。
			rec.EXPECT().Observe(gomock.Any(), "/users/:id", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

			serve(t, rec, serveCfg{
				registerPath: "/users/:id",
				requestPath:  "/users/123?token=secret",
				handler:      okHandler(),
			})
		})

		t.Run("複数回Writeしてもrequestは1件のみ記録される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// 複数回 Write しても Observe は1件のみ。
			rec.EXPECT().Observe(gomock.Any(), gomock.Any(), http.StatusOK, gomock.Any(), gomock.Any()).Times(1)

			serve(t, rec, serveCfg{
				registerPath: "/stream",
				requestPath:  "/stream",
				handler: func(c *echo.Context) error {
					c.Response().WriteHeader(http.StatusOK)
					for range 3 {
						if _, err := c.Response().Write([]byte("chunk")); err != nil {
							return err
						}
					}
					return nil
				},
			})
		})

		t.Run("204応答はAfterフックが発火せず計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗（計測されないことの検証）。

			serve(t, rec, serveCfg{
				registerPath: "/no-content",
				requestPath:  "/no-content",
				handler: func(c *echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				},
			})
		})

		t.Run("route未一致の404ではrouteがunknownでstatus_classが4xxになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			rec.EXPECT().Observe(gomock.Any(), routeUnknown, http.StatusNotFound, "4xx", gomock.Any()).Times(1)

			serve(t, rec, serveCfg{
				requestPath: "/no-such-path",
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("500応答でstatus_classが5xxになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			rec.EXPECT().Observe(http.MethodGet, "/boom", http.StatusInternalServerError, "5xx", gomock.Any()).Times(1)

			serve(t, rec, serveCfg{
				registerPath: "/boom",
				requestPath:  "/boom",
				handler: func(_ *echo.Context) error {
					return echo.NewHTTPError(http.StatusInternalServerError, "")
				},
			})
		})

		t.Run("レスポンスを取り出せない場合は計測せず素通しする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)
			c := e.NewContext(req, httptest.NewRecorder())
			// Echo のレスポンスへ辿れないライタへ差し替える。
			c.SetResponse(httptest.NewRecorder())

			called := false
			handler := Middleware(rec)(func(_ *echo.Context) error {
				called = true
				return nil
			})

			require.NoError(t, handler(c))
			assert.True(t, called, "計測できなくても後続ハンドラは実行される")
		})
	})

	t.Run("運用系パスは計測対象外", func(t *testing.T) {
		t.Parallel()

		t.Run("/metricsは計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			serve(t, rec, serveCfg{
				registerPath: "/metrics",
				requestPath:  "/metrics",
				handler:      okHandler(),
			})
		})

		t.Run("/healthは計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			serve(t, rec, serveCfg{
				registerPath: "/health",
				requestPath:  "/health",
				handler:      okHandler(),
			})
		})

		t.Run("/healthzは計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			serve(t, rec, serveCfg{
				registerPath: "/healthz",
				requestPath:  "/healthz",
				handler:      okHandler(),
			})
		})

		t.Run("/readyは計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			serve(t, rec, serveCfg{
				registerPath: "/ready",
				requestPath:  "/ready",
				handler:      okHandler(),
			})
		})

		t.Run("/versionは計測されない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			rec := mock_redmetrics.NewMockRecorder(ctrl)
			// Observe の EXPECT を設定しない＝呼び出されれば失敗。

			serve(t, rec, serveCfg{
				registerPath: "/version",
				requestPath:  "/version",
				handler:      okHandler(),
			})
		})
	})
}
