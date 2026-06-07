package testecho

import (
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoTestClient_BuildAndServe(t *testing.T) {
	e := echo.New()
	e.GET("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		return c.String(http.StatusOK, "user:"+id)
	})

	t.Parallel()
	t.Run("RoutePatternとPathParamsでリクエストを構築できる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RoutePattern("/users/:id").
			PathParams([]EchoTestParam{{Name: "id", Value: "123"}})

		_, _, c := client.Build()
		assert.Equal(t, "/users/:id", c.Path())
		assert.Equal(t, "123", c.Param("id"))
	})

	t.Run("RequestURLでリクエストを構築できる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RequestURL("/users/456")

		_, _, c := client.Build()
		assert.Equal(t, "/users/456", c.Request().URL.Path)
	})

	t.Run("Serveでレスポンスが取得できる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RequestURL("/users/789")

		rec := client.Serve()
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "user:789", rec.Body.String())
	})
}

func TestEchoTestClient_JSONBody(t *testing.T) {
	e := echo.New()
	t.Parallel()

	t.Run("JSONBodyでContent-Typeがapplication/jsonになる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodPost).
			RoutePattern("/test").
			JSONBody(map[string]string{"foo": "bar"})

		_, _, c := client.Build()
		assert.Equal(t, "application/json", c.Request().Header.Get("Content-Type"))
	})
}

func TestEchoTestClient_HeaderAndAuthBearer(t *testing.T) {
	e := echo.New()
	t.Parallel()

	t.Run("Headerで任意のヘッダーが設定できる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RoutePattern("/test").
			Header("X-Test", "value")

		_, _, c := client.Build()
		assert.Equal(t, "value", c.Request().Header.Get("X-Test"))
	})

	t.Run("AuthBearerでAuthorizationヘッダーが設定できる", func(t *testing.T) {
		token := "abc.def.ghi"
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RoutePattern("/test").
			AuthBearer(token)

		_, _, c := client.Build()
		assert.Equal(t, "Bearer "+token, c.Request().Header.Get("Authorization"))
	})
}

func TestEchoTestClient_QueryParams(t *testing.T) {
	e := echo.New()
	t.Parallel()

	t.Run("QueryParamsでクエリパラメータが設定できる", func(t *testing.T) {
		client := NewEchoTestClient(t, e).
			Method(http.MethodGet).
			RoutePattern("/test").
			QueryParams([]EchoTestParam{{Name: "foo", Value: "bar"}})

		_, _, c := client.Build()
		assert.Equal(t, "bar", c.QueryParam("foo"))
	})
}

func TestEchoTestClient_RawBody(t *testing.T) {
	e := echo.New()
	t.Parallel()

	t.Run("RawBodyでContent-Typeが指定通りになる", func(t *testing.T) {
		body := "raw-body-content"
		contentType := "text/plain"
		client := NewEchoTestClient(t, e).
			Method(http.MethodPost).
			RoutePattern("/test").
			RawBody(strings.NewReader(body), contentType)

		_, _, c := client.Build()
		assert.Equal(t, contentType, c.Request().Header.Get("Content-Type"))
	})

	t.Run("RawBodyでContent-Typeが空の場合、Content-Typeは設定されない", func(t *testing.T) {
		body := "raw-body-content"
		client := NewEchoTestClient(t, e).
			Method(http.MethodPost).
			RoutePattern("/test").
			RawBody(strings.NewReader(body), "")

		_, _, c := client.Build()
		require.Empty(t, c.Request().Header.Get("Content-Type"))
	})
}
