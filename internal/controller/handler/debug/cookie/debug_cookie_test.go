package cookie

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-boilerplate/internal/controller/handler/debug/cookie/gen"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()

	BindHandler(e)

	expectedMethods := []string{
		http.MethodGet,
		http.MethodGet,
		http.MethodGet,
		http.MethodGet,
		http.MethodPost,
		http.MethodDelete,
	}

	expectedPathPrefix := "/debug/cookie"

	actualRoutes := e.Routes()
	for _, r := range actualRoutes {
		require.Contains(t, r.Path, expectedPathPrefix)
	}
	require.Len(t, actualRoutes, len(expectedMethods))

	actualMethods := make([]string, len(actualRoutes))

	for i, r := range actualRoutes {
		actualMethods[i] = r.Method
	}

	require.Len(t, actualMethods, len(expectedMethods))
	for _, method := range expectedMethods {
		require.Contains(t, actualMethods, method)
	}
}

func Test_server_GetDebugCookie(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Cookieが無い場合は空で返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()

			req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.GetDebugCookie(c)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, rec.Code)

			var actual gen.DebugCookieInspectResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &actual))

			require.Empty(t, actual.RawCookieHeader)
			require.Empty(t, actual.Cookies)
		})

		t.Run("Cookieがある場合はRawとMapの両方が返る", func(t *testing.T) {
			t.Parallel()

			cookieHostAccessTokenKey := "__Host-access_token"
			cookieHostAccessTokenValue := "rawtoken"
			cookieThemeKey := "theme"
			cookieThemeValue := "dark"

			cookieVal := cookieHostAccessTokenKey + "=" + cookieHostAccessTokenValue + "; " + cookieThemeKey + "=" + cookieThemeValue

			e := echo.New()
			ctx := context.Background()

			req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
			req.Header.Set("Cookie", cookieVal)

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.GetDebugCookie(c)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, rec.Code)

			var actual gen.DebugCookieInspectResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &actual))

			require.Equal(t, cookieVal, actual.RawCookieHeader)
			require.Equal(t, cookieHostAccessTokenValue, actual.Cookies[cookieHostAccessTokenKey])
			require.Equal(t, cookieThemeValue, actual.Cookies[cookieThemeKey])
		})
	})
}

func Test_server_PostDebugCookie(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookieヘッダが返る", func(t *testing.T) {
			t.Parallel()

			setCookie := "__Host-access_token=rawtoken; Path=/hoge; Domain=example.com; SameSite=None"

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(`{"setCookie":"`+setCookie+`"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.PostDebugCookie(c)
			require.NoError(t, err)

			require.Equal(t, http.StatusNoContent, rec.Code)

			// Echo/httptest の場合、複数 Set-Cookie に備えて Values で確認する
			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			require.Equal(t, setCookie, cookies[0])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正JSONはエラーになる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(`{"setCookie":`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.PostDebugCookie(c)
			require.Error(t, err)
		})
	})
}

func Test_server_DeleteDebugCookie(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name/path未指定の場合はデフォルトで削除Cookieを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.DeleteDebugCookie(c, gen.DeleteDebugCookieParams{})
			require.NoError(t, err)

			require.Equal(t, http.StatusNoContent, rec.Code)

			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			require.Equal(t, "__Host-access_token=; Path=/; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT", cookies[0])
		})

		t.Run("name/path指定の場合は指定値で削除Cookieを返す", func(t *testing.T) {
			t.Parallel()

			name := "__Host-session"
			p := "/hoge"

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodDelete, path+"?name="+name+"&path="+p, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.DeleteDebugCookie(c, gen.DeleteDebugCookieParams{
				Name: &name,
				Path: &p,
			})
			require.NoError(t, err)

			require.Equal(t, http.StatusNoContent, rec.Code)

			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			require.Equal(t, name+"=; Path="+p+"; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT", cookies[0])
		})
	})
}

func Test_server_GetDebugCookieRawCopy(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie/copy"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookie と Content-Type が設定され、ボディが書き込まれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.GetDebugCookieRawCopy(c)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "text/plain; charset=utf-8", rec.Header().Get(echo.HeaderContentType))

			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			require.Equal(t, rawSetCookieSample, cookies[0])

			// ボディが書き込まれていること（先頭だけ確認）
			require.True(t, strings.HasPrefix(rec.Body.String(), "hello-cookie\n"))
		})
	})
}

func Test_server_GetDebugCookieRawStream(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie/stream"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookie と Content-Type が設定され、pingが複数回書き込まれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &server{}
			err := h.GetDebugCookieRawStream(c)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "text/event-stream", rec.Header().Get(echo.HeaderContentType))

			cookies := rec.Header().Values("Set-Cookie")
			require.Len(t, cookies, 1)
			require.Equal(t, rawSetCookieSample, cookies[0])

			// ボディが書き込まれていること（pingが3回）
			body := rec.Body.String()
			require.GreaterOrEqual(t, strings.Count(body, "data: ping\n\n"), 3)
		})
	})
}

func Test_server_GetDebugCookieRawWs(t *testing.T) {
	t.Parallel()

	path := "/debug/cookie/ws"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WebSocketでUpgradeでき、Set-Cookieが付与される", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			h := &server{}
			e.GET(path, h.GetDebugCookieRawWs)

			ts := httptest.NewServer(e)
			t.Cleanup(ts.Close)

			// http://127... -> ws://127...
			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + path

			// WS 接続
			dialer := websocket.Dialer{}
			conn, resp, err := dialer.Dial(wsURL, nil)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			// Upgrade 成功（101）
			require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

			// Set-Cookie が返っている（※ MW rewrite はこのテストでは通らない前提）
			setCookies := resp.Header.Values("Set-Cookie")
			require.NotEmpty(t, setCookies)
			require.Contains(t, setCookies, rawSetCookieSample)

			// echo back 動作（送ったものが返る）
			want := []byte("hello")
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, want))

			mt, got, err := conn.ReadMessage()
			require.NoError(t, err)
			require.Equal(t, websocket.TextMessage, mt)
			require.Equal(t, want, got)
		})
	})
}
