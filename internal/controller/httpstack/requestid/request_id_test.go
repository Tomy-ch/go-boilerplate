package requestid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_getRequestID(t *testing.T) {
	t.Parallel()

	expected := "test-request-id"

	t.Run("X-Request-ID が設定されている場合", func(t *testing.T) {
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Response().Header().Set(echo.HeaderXRequestID, expected)

		actual := GetRequestIDFromResponse(c)
		require.Equal(t, expected, actual)
	})

	t.Run("X-Request-ID が設定されていない場合", func(t *testing.T) {
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		id := GetRequestIDFromResponse(c)
		require.Empty(t, id)
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Middleware())
}
