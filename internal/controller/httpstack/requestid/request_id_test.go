package requestid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetRequestIDFromResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("X-Request-IDが設定されている場合、設定値を返す", func(t *testing.T) {
			t.Parallel()
			expected := "test-request-id"
			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Response().Header().Set(echo.HeaderXRequestID, expected)

			actual := GetRequestIDFromResponse(c)
			assert.Equal(t, expected, actual)
		})

		t.Run("X-Request-IDが設定されていない場合、空文字を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			id := GetRequestIDFromResponse(c)
			assert.Empty(t, id)
		})
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのミドルウェアを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, Middleware())
		})
	})
}
