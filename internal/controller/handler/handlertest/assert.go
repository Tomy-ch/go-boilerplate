// Package handlertest は、ハンドラー用のテストユーティリティを提供します。
package handlertest

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// AssertJSONEqual は、HTTPレスポンスのJSONボディが期待される値と等しいことを検証します。
func AssertJSONEqual[T any](
	t *testing.T, expectedCode int, expectedResponse T, actualResponse *httptest.ResponseRecorder,
) {
	t.Helper()

	require.Equal(t, expectedCode, actualResponse.Code, "HTTPステータスコードが一致しません")

	var actual T
	require.NoError(t, json.Unmarshal(actualResponse.Body.Bytes(), &actual), "JSONレスポンスのデシリアライズに失敗しました")
	require.Equal(t, expectedResponse, actual, "JSONレスポンスの内容が一致しません")
}

// AssertEchoRouterMethods は、EchoのルートのHTTPメソッドが期待通りであることをアサートします。
func AssertEchoRouterMethods(t *testing.T, expectedMethods []string, actualRoute []*echo.Route) {
	t.Helper()
	actualMethods := make([]string, len(actualRoute))
	for i, r := range actualRoute {
		actualMethods[i] = r.Method
	}

	require.Len(t, actualMethods, len(expectedMethods))
	for _, method := range expectedMethods {
		require.Contains(t, actualMethods, method)
	}
}

// AssertEchoRouterPath は、Echoのルートのパスが期待通りであることをアサートします。
func AssertEchoRouterPath(t *testing.T, expectedPath string, actualRoute []*echo.Route) {
	t.Helper()
	for _, r := range actualRoute {
		require.Equal(t, expectedPath, r.Path)
	}
}
