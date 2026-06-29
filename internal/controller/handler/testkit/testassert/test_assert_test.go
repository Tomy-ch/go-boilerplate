package testassert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type testResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func TestAssertJSONEqual(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期待通りのレスポンスの検証ができる", func(t *testing.T) {
			t.Parallel()
			expected := testResponse{Message: "ok", Code: 200}
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(http.StatusOK)
			err := json.NewEncoder(recorder).Encode(expected)
			require.NoError(t, err)

			AssertJSONEqual(t, http.StatusOK, expected, recorder)
		})
	})
}

func TestAssertEchoRouterMethods(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルーティングのメソッドが期待通りの場合、検証が通る", func(t *testing.T) {
			t.Parallel()
			routes := []*echo.Route{
				{Method: "GET"},
				{Method: "POST"},
				{Method: "DELETE"},
			}
			expected := []string{"GET", "POST", "DELETE"}
			AssertEchoRouterMethods(t, expected, routes)
		})
	})
}

func TestAssertEchoRouterPath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべてのルートのパスが期待通りの場合、検証が通る", func(t *testing.T) {
			t.Parallel()
			routes := []*echo.Route{
				{Path: "/users/:id"},
				{Path: "/users/:id"},
			}
			expected := "/users/:id"
			AssertEchoRouterPath(t, expected, routes)
		})
	})
}
