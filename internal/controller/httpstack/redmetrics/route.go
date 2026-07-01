package redmetrics

import "github.com/labstack/echo/v4"

// routeUnknown は、route pattern が取得できない場合の route label の値です。
const routeUnknown = "unknown"

// routeOf は、raw path / query を含まない Echo の route pattern（例: /users/:id）を返します。
// 取得できない場合は unknown に丸めます。
func routeOf(c echo.Context) string {
	route := c.Path()
	if route == "" {
		return routeUnknown
	}
	return route
}
