package testecho

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEchoWithUserRoute() *echo.Echo {
	e := echo.New()
	e.GET("/users/:id", func(c *echo.Context) error {
		return c.String(http.StatusOK, "user:"+c.Param("id"))
	})
	return e
}

func TestEchoTestClient_BuildAndServe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}

func TestEchoTestClient_Build(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestURLモードではルータ解決でパスが設定される", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, newEchoWithUserRoute()).
				Method(http.MethodGet).
				RequestURL("/users/456")

			req, rec, c := client.Build()
			require.NotNil(t, req)
			require.NotNil(t, rec)
			assert.Equal(t, "/users/:id", c.Path())
			assert.Equal(t, "456", c.Param("id"))
		})

		t.Run("RoutePatternモードではSetPathとPathParamsが設定される", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/users/:id").
				PathParams([]EchoTestParam{{Name: "id", Value: "123"}})

			_, _, c := client.Build()
			assert.Equal(t, "/users/:id", c.Path())
			assert.Equal(t, "123", c.Param("id"))
		})

		t.Run("RoutePatternモードでPathParamsが無ければパラメータは設定されない", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/health")

			_, _, c := client.Build()
			assert.Equal(t, "/health", c.Path())
			assert.Empty(t, c.PathValues())
		})
	})
}

func TestEchoTestClient_buildRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダとクエリパラメータが反映される", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/test").
				Header("X-Test", "value").
				QueryParams([]EchoTestParam{{Name: "foo", Value: "bar"}})

			req, rec := client.buildRequest()
			require.NotNil(t, req)
			require.NotNil(t, rec)
			assert.Equal(t, "value", req.Header.Get("X-Test"))
			assert.Equal(t, "bar", req.URL.Query().Get("foo"))
			assert.Equal(t, "/test", req.URL.Path)
		})

		t.Run("ボディが設定される", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodPost).
				RoutePattern("/test").
				RawBody(strings.NewReader("payload"), "text/plain")

			req, _ := client.buildRequest()
			got, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Equal(t, "payload", string(got))
		})
	})
}

func TestEchoTestClient_resolveTarget(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestURLのみ指定でURLを返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())
			client.RequestURL("/users/1")

			got, err := client.resolveTarget()
			require.NoError(t, err)
			assert.Equal(t, "/users/1", got)
		})

		t.Run("RoutePatternのみ指定でパターンを返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())
			client.RoutePattern("/users/:id")

			got, err := client.resolveTarget()
			require.NoError(t, err)
			assert.Equal(t, "/users/:id", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定はerrTargetUnsetを返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())

			got, err := client.resolveTarget()
			require.ErrorIs(t, err, errTargetUnset)
			assert.Empty(t, got)
		})

		t.Run("RequestURLとRoutePattern併用はerrModeConflictを返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())
			client.RequestURL("/users/1").RoutePattern("/users/:id")

			got, err := client.resolveTarget()
			require.ErrorIs(t, err, errModeConflict)
			assert.Empty(t, got)
		})

		t.Run("RequestURLとPathParams併用はerrModeConflictを返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New())
			client.RequestURL("/users/1").PathParams([]EchoTestParam{{Name: "id", Value: "1"}})

			got, err := client.resolveTarget()
			require.ErrorIs(t, err, errModeConflict)
			assert.Empty(t, got)
		})
	})
}

func TestEchoTestClient_WithAppErrorHandler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラーを返すハンドラがアプリ標準のJSONエラー応答になる", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			e.GET("/boom", func(_ *echo.Context) error {
				return xerrors.New("boom")
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
	})
}

func TestEchoTestClient_JSONBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}

func TestEchoTestClient_HeaderAndAuthBearer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}

func TestEchoTestClient_QueryParams(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}

func TestEchoTestClient_RawBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
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
	})
}
