// Package controller は、コントローラ層に関連するコードを提供します。
package controller

import (
	"net/url"

	"github.com/labstack/echo/v4"
)

// ExtractPathParams は、Echoコンテキストからパスパラメータを抽出します。
func ExtractPathParams(c echo.Context) map[string]string {
	m := make(map[string]string, len(c.ParamNames()))
	for _, name := range c.ParamNames() {
		m[name] = c.Param(name)
	}
	return m
}

// ExtractQueryParams は、Echoコンテキストからクエリパラメータを抽出します。
func ExtractQueryParams(c echo.Context) map[string][]string {
	return cloneValues(c.Request().URL.Query())
}

// cloneValues は、引数についてディープコピーを作成します。
func cloneValues(v url.Values) map[string][]string {
	if v == nil {
		return nil
	}
	m := make(map[string][]string, len(v))
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		m[k] = cp
	}
	return m
}
