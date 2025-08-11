package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_getRequestID(t *testing.T) {
	t.Parallel()

	t.Run("X-Request-ID が設定されている場合", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderXRequestID, "test-request-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		id := GetRequestIDFromResponse(c)
		require.Equal(t, "test-request-id", id)
	})

	t.Run("X-Request-ID が設定されていない場合", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		id := GetRequestIDFromResponse(c)
		require.Empty(t, id)
	})
}
