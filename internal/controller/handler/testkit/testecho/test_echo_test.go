package testecho

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEchoWithUserRoute() *echo.Echo {
	e := echo.New()
	e.GET("/users/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "user:"+c.Param("id"))
	})
	return e
}

func TestEchoTestClient_BuildAndServe(t *testing.T) {
	t.Parallel()

	t.Run("RoutePatternとPathParamsでリクエストを構築できる", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, newEchoWithUserRoute()).
			Method(http.MethodGet).
			RoutePattern("/users/:id").
			PathParams([]EchoTestParam{{Name: "id", Value: "123"}})

		_, _, c := client.Build()
		assert.Equal(t, "/users/:id", c.Path())
		assert.Equal(t, "123", c.Param("id"))
	})

	t.Run("RequestURLでリクエストを構築できる", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, newEchoWithUserRoute()).
			Method(http.MethodGet).
			RequestURL("/users/456")

		_, _, c := client.Build()
		assert.Equal(t, "/users/456", c.Request().URL.Path)
	})

	t.Run("Serveでレスポンスが取得できる", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, newEchoWithUserRoute()).
			Method(http.MethodGet).
			RequestURL("/users/789")

		rec := client.Serve()
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "user:789", rec.Body.String())
	})
}

func TestEchoTestClient_resolveTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(c *EchoTestClient)
		want    string
		wantErr error
	}{
		{
			name:    "RequestURLのみ指定でURLを返す",
			setup:   func(c *EchoTestClient) { c.RequestURL("/users/1") },
			want:    "/users/1",
			wantErr: nil,
		},
		{
			name:    "RoutePatternのみ指定でパターンを返す",
			setup:   func(c *EchoTestClient) { c.RoutePattern("/users/:id") },
			want:    "/users/:id",
			wantErr: nil,
		},
		{
			name:    "未指定はerrTargetUnset",
			setup:   func(_ *EchoTestClient) {},
			want:    "",
			wantErr: errTargetUnset,
		},
		{
			name: "RequestURLとRoutePattern併用はerrModeConflict",
			setup: func(c *EchoTestClient) {
				c.RequestURL("/users/1").RoutePattern("/users/:id")
			},
			want:    "",
			wantErr: errModeConflict,
		},
		{
			name: "RequestURLとPathParams併用はerrModeConflict",
			setup: func(c *EchoTestClient) {
				c.RequestURL("/users/1").PathParams([]EchoTestParam{{Name: "id", Value: "1"}})
			},
			want:    "",
			wantErr: errModeConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())
			tt.setup(client)

			got, err := client.resolveTarget()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEchoTestClient_WithAppErrorHandler(t *testing.T) {
	t.Parallel()

	t.Run("エラーを返すハンドラがアプリ標準のJSONエラー応答になる", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		e.GET("/boom", func(_ echo.Context) error {
			return errors.New("boom")
		})

		rec := NewEchoTestClient(t, e).
			WithAppErrorHandler().
			Method(http.MethodGet).
			RequestURL("/boom").
			Serve()

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.NotEmpty(t, body["code"])
	})
}

func TestEchoTestClient_JSONBody(t *testing.T) {
	t.Parallel()

	t.Run("JSONBodyでContent-Typeとボディが設定される", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodPost).
			RoutePattern("/test").
			JSONBody(map[string]string{"foo": "bar"})

		_, _, c := client.Build()
		assert.Equal(t, "application/json", c.Request().Header.Get("Content-Type"))

		got, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"foo":"bar"}`, string(got))
	})
}

func TestEchoTestClient_HeaderAndAuthBearer(t *testing.T) {
	t.Parallel()

	t.Run("Headerで任意のヘッダーが設定できる", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodGet).
			RoutePattern("/test").
			Header("X-Test", "value")

		_, _, c := client.Build()
		assert.Equal(t, "value", c.Request().Header.Get("X-Test"))
	})

	t.Run("AuthBearerでAuthorizationヘッダーが設定できる", func(t *testing.T) {
		t.Parallel()
		token := "abc.def.ghi"
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodGet).
			RoutePattern("/test").
			AuthBearer(token)

		_, _, c := client.Build()
		assert.Equal(t, "Bearer "+token, c.Request().Header.Get("Authorization"))
	})
}

func TestEchoTestClient_QueryParams(t *testing.T) {
	t.Parallel()

	t.Run("QueryParamsでクエリパラメータが設定できる", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodGet).
			RoutePattern("/test").
			QueryParams([]EchoTestParam{{Name: "foo", Value: "bar"}})

		_, _, c := client.Build()
		assert.Equal(t, "bar", c.QueryParam("foo"))
	})
}

func TestEchoTestClient_RawBody(t *testing.T) {
	t.Parallel()

	t.Run("RawBodyでContent-Typeとボディが指定通りになる", func(t *testing.T) {
		t.Parallel()
		body := "raw-body-content"
		contentType := "text/plain"
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodPost).
			RoutePattern("/test").
			RawBody(strings.NewReader(body), contentType)

		_, _, c := client.Build()
		assert.Equal(t, contentType, c.Request().Header.Get("Content-Type"))

		got, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
	})

	t.Run("RawBodyでContent-Typeが空の場合、Content-Typeは設定されない", func(t *testing.T) {
		t.Parallel()
		client := NewEchoTestClient(t, echo.New()).
			Method(http.MethodPost).
			RoutePattern("/test").
			RawBody(strings.NewReader("raw-body-content"), "")

		_, _, c := client.Build()
		assert.Empty(t, c.Request().Header.Get("Content-Type"))
	})
}
