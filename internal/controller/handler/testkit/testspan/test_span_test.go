package testspan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartTestSpanForEcho(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リクエストにspanが付与されSpanID/TraceIDが非空になる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			c := e.NewContext(req, httptest.NewRecorder())

			nc, end := StartTestSpanForEcho(t, c)
			t.Cleanup(end)

			require.NotNil(t, end)
			assert.Equal(t, c, nc)

			sc := observability.ExtractTraceContext(nc.Request().Context())
			assert.NotEmpty(t, sc.SpanID())
			assert.NotEmpty(t, sc.TraceID())
		})
	})
}
