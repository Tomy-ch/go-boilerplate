// Package metrics は、メトリクス関連のハンドラを提供します。
package metrics

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	echomw "github.com/labstack/echo/v4/middleware"
)

func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
	e.GET("/metrics",
		echo.WrapHandler(promhttp.Handler()),
		echomw.BasicAuth(bav),
	)
}
