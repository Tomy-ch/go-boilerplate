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

	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{name: "未設定(空)はtrue", ct: "", want: true},
		{name: "text/htmlはtrue", ct: echo.MIMETextHTML, want: true},
		{name: "text/html;charset付きでもtrue", ct: echo.MIMETextHTML + "; charset=iso-8859-1", want: true},
		{name: "application/jsonはfalse", ct: echo.MIMEApplicationJSON, want: false},
		{name: "application/xmlはfalse", ct: echo.MIMEApplicationXML, want: false},
		{name: "text/plainはfalse", ct: echo.MIMETextPlain, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldForceJSON(tt.ct))
		})
	}
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

	t.Run("ヘッダが空の場合は強制される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, echo.MIMEApplicationJSON, got)
	})

	t.Run("text/html の場合は強制される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, echo.MIMEApplicationJSON, got)
	})

	t.Run("application/json の場合は変更されない", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, echo.MIMEApplicationJSON, got)
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Middleware())
}

// TestMiddleware_overWire は、実 HTTP 経路（commit 済みレスポンス）でも Content-Type が
// 上書きされることを検証する。recorder の生ヘッダマップでは commit 後の挙動を検出できないため
// httptest.NewServer を用いてワイヤ上の最終ヘッダを確認する。
func TestMiddleware_overWire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler echo.HandlerFunc
		wantCT  string
	}{
		{
			name:    "HTMLボディは application/json へ上書きされる",
			handler: func(c echo.Context) error { return c.HTML(http.StatusOK, "<p>hi</p>") },
			wantCT:  echo.MIMEApplicationJSON,
		},
		{
			name:    "Content-Type未設定のボディも application/json へ上書きされる",
			handler: func(c echo.Context) error { return c.Blob(http.StatusOK, "", []byte("x")) },
			wantCT:  echo.MIMEApplicationJSON,
		},
		{
			name:    "application/json は変更されない",
			handler: func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"k": "v"}) },
			wantCT:  echo.MIMEApplicationJSON,
		},
		{
			name:    "text/plain は変更されない",
			handler: func(c echo.Context) error { return c.String(http.StatusOK, "plain") },
			wantCT:  echo.MIMETextPlainCharsetUTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Use(Middleware())
			e.GET("/t", tt.handler)

			srv := httptest.NewServer(e)
			defer srv.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/t", nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantCT, resp.Header.Get(echo.HeaderContentType))
		})
	}
}
