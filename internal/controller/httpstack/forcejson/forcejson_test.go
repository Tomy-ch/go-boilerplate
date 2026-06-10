package forcejson

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_shouldForceJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未設定(空)の場合、trueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, shouldForceJSON(""))
		})

		t.Run("text/htmlの場合、trueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, shouldForceJSON(echo.MIMETextHTML))
		})

		t.Run("text/html;charset付きでもtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, shouldForceJSON(echo.MIMETextHTML+"; charset=iso-8859-1"))
		})

		t.Run("application/jsonの場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, shouldForceJSON(echo.MIMEApplicationJSON))
		})

		t.Run("application/xmlの場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, shouldForceJSON(echo.MIMEApplicationXML))
		})

		t.Run("text/plainの場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, shouldForceJSON(echo.MIMETextPlain))
		})
	})
}

func Test_ensureJSONContentType(t *testing.T) {
	t.Parallel()

	newCtx := func() echo.Context {
		e := echo.New()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダが空の場合は強制される", func(t *testing.T) {
			t.Parallel()
			c := newCtx()

			ensureJSONContentType(c)

			got := c.Response().Header().Get(echo.HeaderContentType)
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})

		t.Run("text/htmlの場合は強制される", func(t *testing.T) {
			t.Parallel()
			c := newCtx()

			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
			ensureJSONContentType(c)

			got := c.Response().Header().Get(echo.HeaderContentType)
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})

		t.Run("application/jsonの場合は変更されない", func(t *testing.T) {
			t.Parallel()
			c := newCtx()

			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			ensureJSONContentType(c)

			got := c.Response().Header().Get(echo.HeaderContentType)
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのミドルウェアを返す", func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, Middleware())
		})
	})
}

// TestMiddleware_overWire は、実 HTTP 経路（commit 済みレスポンス）でも Content-Type が
// 上書きされることを検証する。recorder の生ヘッダマップでは commit 後の挙動を検出できないため
// httptest.NewServer を用いてワイヤ上の最終ヘッダを確認する。
func TestMiddleware_overWire(t *testing.T) {
	t.Parallel()

	exec := func(t *testing.T, handler echo.HandlerFunc) string {
		t.Helper()
		e := echo.New()
		e.Use(Middleware())
		e.GET("/t", handler)

		srv := httptest.NewServer(e)
		t.Cleanup(srv.Close)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/t", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.Header.Get(echo.HeaderContentType)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HTMLボディはapplication/jsonへ上書きされる", func(t *testing.T) {
			t.Parallel()
			got := exec(t, func(c echo.Context) error { return c.HTML(http.StatusOK, "<p>hi</p>") })
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})

		t.Run("Content-Type未設定のボディもapplication/jsonへ上書きされる", func(t *testing.T) {
			t.Parallel()
			got := exec(t, func(c echo.Context) error { return c.Blob(http.StatusOK, "", []byte("x")) })
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})

		t.Run("application/jsonは変更されない", func(t *testing.T) {
			t.Parallel()
			got := exec(t, func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"k": "v"}) })
			assert.Equal(t, echo.MIMEApplicationJSON, got)
		})

		t.Run("text/plainは変更されない", func(t *testing.T) {
			t.Parallel()
			got := exec(t, func(c echo.Context) error { return c.String(http.StatusOK, "plain") })
			assert.Equal(t, echo.MIMETextPlainCharsetUTF8, got)
		})
	})
}
