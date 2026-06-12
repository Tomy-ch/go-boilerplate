package uri

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	exec := func(t *testing.T, path string) (string, int) {
		t.Helper()
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var seenPath string
		handler := Middleware()(func(c echo.Context) error {
			seenPath = c.Request().URL.Path
			return c.NoContent(http.StatusOK)
		})

		require.NoError(t, handler(c))
		return seenPath, rec.Code
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("末尾スラッシュがある場合、除去されて次のハンドラに渡る", func(t *testing.T) {
			t.Parallel()
			seenPath, status := exec(t, "/foo/")
			assert.Equal(t, "/foo", seenPath)
			assert.Equal(t, http.StatusOK, status)
		})

		t.Run("末尾スラッシュが無い場合、パスはそのまま渡る", func(t *testing.T) {
			t.Parallel()
			seenPath, status := exec(t, "/foo")
			assert.Equal(t, "/foo", seenPath)
			assert.Equal(t, http.StatusOK, status)
		})
	})
}
