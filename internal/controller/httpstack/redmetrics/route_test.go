package redmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func newContext(t *testing.T, path string) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)
	c.SetPath(path)
	return c
}

func TestRouteOf(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("route_patternが設定されていればそのまま返す", func(t *testing.T) {
			t.Parallel()
			c := newContext(t, "/users/:id")
			assert.Equal(t, "/users/:id", routeOf(c))
		})

		t.Run("routeが空ならunknownを返す", func(t *testing.T) {
			t.Parallel()
			c := newContext(t, "")
			assert.Equal(t, routeUnknown, routeOf(c))
		})
	})
}
