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

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "末尾スラッシュがある場合_除去されて次のハンドラに渡る",
			path:     "/foo/",
			wantPath: "/foo",
		},
		{
			name:     "末尾スラッシュが無い場合_パスはそのまま",
			path:     "/foo",
			wantPath: "/foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var seenPath string
			handler := Middleware()(func(c echo.Context) error {
				seenPath = c.Request().URL.Path
				return c.NoContent(http.StatusOK)
			})

			require.NoError(t, handler(c))
			assert.Equal(t, tt.wantPath, seenPath)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
