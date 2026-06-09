package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ExtractPathParams(t *testing.T) {
	t.Parallel()

	e := echo.New()
	newCtx := func() echo.Context {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/users/123/books/abc", nil)
		return e.NewContext(req, httptest.NewRecorder())
	}

	t.Run("パスパラメータがない場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		got := ExtractPathParams(newCtx())
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("パスパラメータがある場合は全て抽出される", func(t *testing.T) {
		t.Parallel()
		c := newCtx()
		c.SetParamNames("user_id", "book_id")
		c.SetParamValues("123", "abc")

		got := ExtractPathParams(c)
		require.Len(t, got, 2)
		assert.Equal(t, "123", got["user_id"])
		assert.Equal(t, "abc", got["book_id"])
	})
}

func Test_ExtractQueryParams(t *testing.T) {
	t.Parallel()

	e := echo.New()

	t.Run("クエリが無い場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got := ExtractQueryParams(c)
		// 実装上は空の map を返すので長さが0であることを確認
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("複数値クエリを正しくコピーして返す", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/path?foo=1&foo=2&bar=x", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got := ExtractQueryParams(c)
		require.NotNil(t, got)
		assert.Equal(t, []string{"1", "2"}, got["foo"])
		assert.Equal(t, []string{"x"}, got["bar"])
	})
}

func Test_BuildHTTPRequestLogInput(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/users/123?q=v", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("User-Agent", "test-agent")
	req.Host = "example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:id")
	c.SetParamNames("id")
	c.SetParamValues("123")

	got := BuildHTTPRequestLogInput(c)

	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/users/:id", got.Path)
	assert.Equal(t, "/users/123?q=v", got.URI)
	assert.Equal(t, "example.com", got.Host)
	assert.Equal(t, "http", got.Scheme)
	assert.Equal(t, "test-agent", got.UserAgent)
	assert.Equal(t, echo.MIMEApplicationJSON, got.ContentType)
	assert.Equal(t, map[string]string{"id": "123"}, got.PathParams)
	assert.Equal(t, []string{"v"}, got.QueryParams["q"])
	assert.False(t, got.EventAt.IsZero())
}

func Test_Recovered(t *testing.T) {
	t.Parallel()

	e := echo.New()
	newCtx := func() echo.Context {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		return e.NewContext(req, httptest.NewRecorder())
	}

	t.Run("未設定なら false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsRecovered(newCtx()))
	})

	t.Run("MarkRecovered 後は true", func(t *testing.T) {
		t.Parallel()
		c := newCtx()
		MarkRecovered(c)
		assert.True(t, IsRecovered(c))
	})
}
