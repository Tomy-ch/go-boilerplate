package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ミドルウェアが生成される", func(t *testing.T) {
			t.Parallel()

			mw := Middleware("test-service")
			assert.NotNil(t, mw)
		})
	})
}

func TestPassthroughMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("後段ハンドラへ素通しし200を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Use(PassthroughMiddleware())
			e.GET("/", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	})
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録ルートは404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Use(Middleware("test-service"))

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	})
}
