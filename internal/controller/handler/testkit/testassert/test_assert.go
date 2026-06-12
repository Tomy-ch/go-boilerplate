// Package testassert は、テスト用のアサーションヘルパーを提供します。
package testassert

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertJSONEqual は、HTTPレスポンスのJSONボディが期待される値と等しいことを検証します。
func AssertJSONEqual[T any](
	t *testing.T, expectedCode int, expectedResponse T, actualResponse *httptest.ResponseRecorder,
) {
	t.Helper()

	assert.Equal(t, expectedCode, actualResponse.Code, "HTTPステータスコードが一致しません")

	var actual T
	require.NoError(t, json.Unmarshal(actualResponse.Body.Bytes(), &actual), "JSONレスポンスのデシリアライズに失敗しました")
	assert.Equal(t, expectedResponse, actual, "JSONレスポンスの内容が一致しません")
}

// AssertEchoRouterMethods は、EchoのルートのHTTPメソッド集合が期待通りであることをアサートします。
func AssertEchoRouterMethods(t *testing.T, expectedMethods []string, actualRoute []*echo.Route) {
	t.Helper()
	actualMethods := make([]string, len(actualRoute))
	for i, r := range actualRoute {
		actualMethods[i] = r.Method
	}

	assert.ElementsMatch(t, expectedMethods, actualMethods)
}

// AssertEchoRouterPath は、Echoのルートのパスが期待通りであることをアサートします。
func AssertEchoRouterPath(t *testing.T, expectedPath string, actualRoute []*echo.Route) {
	t.Helper()
	require.NotEmpty(t, actualRoute, "ルートが登録されていません")
	for _, r := range actualRoute {
		assert.Equal(t, expectedPath, r.Path)
	}
}
