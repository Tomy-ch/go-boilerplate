package forcejson

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isBlacklistedContentType(t *testing.T) {
	t.Run("何も設定されていない場合はtrueを返す", func(t *testing.T) {
		assert.True(t, isBlacklistedContentType(""))
	})
	t.Run("text/htmlの場合はtrueを返す", func(t *testing.T) {
		assert.True(t, isBlacklistedContentType(echo.MIMETextHTML))
	})
	t.Run("application/jsonの場合はfalseを返す", func(t *testing.T) {
		assert.False(t, isBlacklistedContentType(echo.MIMEApplicationJSON))
	})
	t.Run("application/xmlの場合はfalseを返す", func(t *testing.T) {
		assert.False(t, isBlacklistedContentType(echo.MIMEApplicationXML))
	})
}

func Test_jsonContentTypeWithCharset(t *testing.T) {
	t.Parallel()

	t.Run("application/json; charset=UTF-8 を返す", func(t *testing.T) {
		expected := echo.MIMEApplicationJSON + "; " + charsetUTF8
		actual := jsonContentTypeWithCharset()
		assert.Equal(t, expected, actual)
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

	t.Run("ヘッダが空の場合は強制される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, jsonContentTypeWithCharset(), got)
	})

	t.Run("text/html の場合は強制される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, jsonContentTypeWithCharset(), got)
	})

	t.Run("text/html; charset が付与されていても強制される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML+"; charset=iso-8859-1")
		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, jsonContentTypeWithCharset(), got)
	})

	t.Run("application/json の場合は変更されない", func(t *testing.T) {
		t.Parallel()
		c := newCtx()

		orig := echo.MIMEApplicationJSON
		c.Response().Header().Set(echo.HeaderContentType, orig)
		ensureJSONContentType(c)

		got := c.Response().Header().Get(echo.HeaderContentType)
		assert.Equal(t, orig, got)
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	require.NotNil(t, Middleware())
}

func Test_forceJSONContentTypeMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("ヘッダが空の場合は強制される", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		e.Use(Middleware())

		e.GET("/test-empty", func(c echo.Context) error {
			c.Response().WriteHeader(http.StatusOK)
			return nil
		})

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test-empty", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		got := rec.Header().Get(echo.HeaderContentType)
		assert.Equal(t, jsonContentTypeWithCharset(), got)
	})

	t.Run("text/html の場合は強制される", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		e.Use(Middleware())

		e.GET("/test-html", func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
			c.Response().WriteHeader(http.StatusOK)
			return nil
		})

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test-html", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		got := rec.Header().Get(echo.HeaderContentType)
		assert.Equal(t, jsonContentTypeWithCharset(), got)
	})

	t.Run("application/json の場合は変更されない", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		e.Use(Middleware())

		e.GET("/test-json", func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c.Response().WriteHeader(http.StatusOK)
			return nil
		})

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test-json", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		got := rec.Header().Get(echo.HeaderContentType)
		assert.Equal(t, echo.MIMEApplicationJSON, got)
	})
}
