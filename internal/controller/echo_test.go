package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_ExtractPathParams(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/123/books/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	t.Run("パスパラメータがない場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		// 新しいコンテキストでは ParamNames 未設定 -> 空
		got := ExtractPathParams(c)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("パスパラメータがある場合は全て抽出される", func(t *testing.T) {
		t.Parallel()
		// 名前と値をセット
		c.SetParamNames("user_id", "book_id")
		c.SetParamValues("123", "abc")

		got := ExtractPathParams(c)
		require.Len(t, got, 2)
		require.Equal(t, "123", got["user_id"])
		require.Equal(t, "abc", got["book_id"])
	})
}

func Test_ExtractQueryParams(t *testing.T) {
	t.Parallel()

	e := echo.New()

	t.Run("クエリが無い場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/path", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got := ExtractQueryParams(c)
		// 実装上は空の map を返すので長さが0であることを確認
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("複数値クエリを正しくコピーして返す", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/path?foo=1&foo=2&bar=x", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got := ExtractQueryParams(c)
		require.NotNil(t, got)
		require.Equal(t, []string{"1", "2"}, got["foo"])
		require.Equal(t, []string{"x"}, got["bar"])

		// 返されたマップは独立コピーであることを確認（元の URL 値を変更しても影響しない）
		// 元の values にアクセス
		orig := c.Request().URL.Query()
		orig.Set("foo", "9")
		// got は以前の値のまま
		require.Equal(t, []string{"1", "2"}, got["foo"])
	})
}

func Test_cloneValues(t *testing.T) {
	t.Parallel()

	t.Run("nil を渡すと nil を返す", func(t *testing.T) {
		t.Parallel()
		var v url.Values
		got := cloneValues(v)
		require.Nil(t, got)
	})

	t.Run("コピーはディープコピーであること", func(t *testing.T) {
		t.Parallel()
		v := url.Values{}
		v.Add("a", "1")
		v.Add("a", "2")

		cp := cloneValues(v)
		require.Equal(t, []string{"1", "2"}, cp["a"])

		// 元を変更してもコピーに影響しない
		v.Set("a", "9")
		require.Equal(t, []string{"1", "2"}, cp["a"])
	})
}
