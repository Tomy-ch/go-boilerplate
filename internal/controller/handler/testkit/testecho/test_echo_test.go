package testecho

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestNewEchoTestClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したEchoを保持し設定前のビルダは空で初期化される", func(t *testing.T) {
			t.Parallel()
			e := echo.New()

			client := NewEchoTestClient(t, e)

			assert.Same(t, e, client.e)
			// headers は Header()/AuthBearer() が直接 Set できるよう初期化済みであること。
			require.NotNil(t, client.headers)
			assert.Empty(t, client.headers)
			assert.Empty(t, client.method)
			assert.Empty(t, client.routePattern)
			assert.Empty(t, client.requestURL)
			assert.Nil(t, client.body)
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

func TestEchoTestClient_Method(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したメソッドがリクエストに反映されチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).RoutePattern("/test")

			assert.Same(t, client, client.Method(http.MethodPatch))

			req, _ := client.buildRequest()
			assert.Equal(t, http.MethodPatch, req.Method)
		})
	})
}

func TestEchoTestClient_RoutePattern(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したパターンがコンテキストのパスになりチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).Method(http.MethodGet)

			assert.Same(t, client, client.RoutePattern("/users/:id"))

			req, _, c := client.Build()
			assert.Equal(t, "/users/:id", c.Path())
			// ルータ登録に依らず、パターン文字列がそのままリクエスト先になる。
			assert.Equal(t, "/users/:id", req.URL.Path)
		})
	})
}

func TestEchoTestClient_RequestURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリを含むURLがそのままリクエスト先になりチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, newEchoWithUserRoute()).Method(http.MethodGet)

			assert.Same(t, client, client.RequestURL("/users/456?limit=10"))

			req, _, c := client.Build()
			assert.Equal(t, "/users/456", req.URL.Path)
			assert.Equal(t, "10", req.URL.Query().Get("limit"))
			// ルータ解決を通るため、パスパラメータを渡さなくても解決済みの経路情報が入る。
			assert.Equal(t, "/users/:id", c.Path())
			assert.Equal(t, "456", c.Param("id"))
		})
	})
}

func TestEchoTestClient_PathParams(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したパスパラメータがコンテキストから引けチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/users/:id")

			assert.Same(t, client, client.PathParams([]EchoTestParam{{Name: "id", Value: "123"}}))

			_, _, c := client.Build()
			assert.Equal(t, "123", c.Param("id"))
		})

		t.Run("複数のパスパラメータをすべて設定する", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/orgs/:orgId/users/:id").
				PathParams([]EchoTestParam{{Name: "orgId", Value: "o-1"}, {Name: "id", Value: "123"}})

			_, _, c := client.Build()
			assert.Equal(t, "o-1", c.Param("orgId"))
			assert.Equal(t, "123", c.Param("id"))
		})
	})
}

func TestEchoTestClient_Header(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したヘッダがリクエストに反映されチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/test")

			assert.Same(t, client, client.Header("X-Test", "value"))

			_, _, c := client.Build()
			assert.Equal(t, "value", c.Request().Header.Get("X-Test"))
		})

		t.Run("同じキーを再設定すると後の値で上書きされる", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/test").
				Header("X-Test", "first").
				Header("X-Test", "second")

			_, _, c := client.Build()
			assert.Equal(t, []string{"second"}, c.Request().Header.Values("X-Test"))
		})
	})
}

func TestEchoTestClient_AuthBearer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("BearerプレフィックスつきでAuthorizationヘッダを設定しチェーン用に自身を返す", func(t *testing.T) {
			t.Parallel()
			client := NewEchoTestClient(t, echo.New()).
				Method(http.MethodGet).
				RoutePattern("/test")

			assert.Same(t, client, client.AuthBearer("abc.def.ghi"))

			_, _, c := client.Build()
			assert.Equal(t, "Bearer abc.def.ghi", c.Request().Header.Get(echo.HeaderAuthorization))
		})
	})
}

func TestEchoTestClient_Serve(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みルートへ送信しハンドラのレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			rec := NewEchoTestClient(t, newEchoWithUserRoute()).
				Method(http.MethodGet).
				RequestURL("/users/789").
				Serve()

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "user:789", rec.Body.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録ルートへ送信するとEchoの404が返る", func(t *testing.T) {
			t.Parallel()

			rec := NewEchoTestClient(t, newEchoWithUserRoute()).
				Method(http.MethodGet).
				RequestURL("/no/such/path").
				Serve()

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	})
}

func Test_newTestDetailPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実specからdetails公開をopt-inしたoperationを許可するポリシーを返す", func(t *testing.T) {
			t.Parallel()

			policy := newTestDetailPolicy(t)

			require.NotNil(t, policy)
			// 常に拒否するスタブではなく、実 spec の opt-in 情報を持つポリシーであること。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/users", nil)
			assert.True(t, policy.Allows(req))
		})
	})
}

func Test_newTestAllowPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実specからパスごとの許可メソッドを解決するポリシーを返す", func(t *testing.T) {
			t.Parallel()

			policy := newTestAllowPolicy(t)

			require.NotNil(t, policy)
			// 常に空文字を返すスタブではなく、実 spec のメソッド情報を持つポリシーであること。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/prefectures", nil)
			assert.Equal(t, "OPTIONS, GET", policy.Allow(req))
		})
	})
}
