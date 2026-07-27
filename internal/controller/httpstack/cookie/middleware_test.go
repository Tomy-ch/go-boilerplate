package cookie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/server"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのミドルウェアを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, Middleware(&SecurityCookie{}))
		})
	})
}

func Test_secureCookieMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookieにセキュリティ属性が付与される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			secCfg := config.NewSecureCookieConfig(cfg)
			secCookie := NewSecurityCookie(secCfg)

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			next := func(c *echo.Context) error {
				c.Response().Header().Add("Set-Cookie", "sessionid=abc; Path=/; HttpOnly")
				return c.String(http.StatusOK, "ok")
			}

			handler := secureCookieMiddleware(secCookie)(next)
			require.NoError(t, handler(c))

			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			assert.Contains(t, cookies[0], "SameSite=Strict")
			assert.Contains(t, cookies[0], "Domain=localhost")
			assert.Contains(t, cookies[0], "Secure")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nextがエラーを返してもResponseWriterは復元されない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := secureCookieMiddleware(&SecurityCookie{applyToAll: true})(func(_ *echo.Context) error {
				return echo.NewHTTPError(http.StatusInternalServerError, "")
			})
			require.Error(t, handler(c))

			_, ok := server.ResponseOf(c).ResponseWriter.(*cookieRewriteWriter)
			assert.True(t, ok)
		})

		t.Run("レスポンスを取り出せない場合は書き換えず素通しする", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// Echo のレスポンスへ辿れないライタへ差し替える。
			c.SetResponse(rec)

			called := false
			handler := secureCookieMiddleware(&SecurityCookie{applyToAll: true})(func(c *echo.Context) error {
				called = true
				c.Response().Header().Add("Set-Cookie", "sessionid=abc; Path=/")
				return nil
			})
			require.NoError(t, handler(c))

			assert.True(t, called, "書き換えできなくても後続ハンドラは実行される")
			assert.Equal(t, "sessionid=abc; Path=/", rec.Header().Get("Set-Cookie"))
		})
	})
}
