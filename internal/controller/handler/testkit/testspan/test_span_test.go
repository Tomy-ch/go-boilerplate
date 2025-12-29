package testspan

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestStartTestSpanForEcho(t *testing.T) {
	t.Parallel()

	t.Run("グローバルトレーサープロバイダが設定されている場合、span が有効になる", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		setupCtx, setupSpan := observability.NewStubSpanContext(t)
		defer setupSpan()
		req = req.WithContext(setupCtx)

		c := e.NewContext(req, rec)

		nc, end := StartTestSpanForEcho(t, c)
		defer end()
		require.NotNil(t, nc)
		require.NotNil(t, end)

		sc := observability.ExtractTraceContext(c.Request().Context())
		require.NotEmpty(t, sc.SpanID())
		require.NotEmpty(t, sc.TraceID())

		end()
	})

	t.Run("グローバルトレーサープロバイダが無い（デフォルト）場合でも呼び出し可能", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/no", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nc, end := StartTestSpanForEcho(t, c)
		require.NotNil(t, nc)
		require.NotNil(t, end)

		_ = observability.ExtractTraceContext(c.Request().Context())
		end()
	})
}
