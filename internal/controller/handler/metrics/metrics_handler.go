// Package metrics は、メトリクス関連のハンドラを提供します。
package metrics

import (
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	echomw "github.com/labstack/echo/v5/middleware"
)

// BindHandler は、Prometheusメトリクスエンドポイント（/metrics）をEchoに登録します。
// Basic認証バリデータを受け取り、エンドポイントへのアクセスを保護します。
func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
	e.GET("/metrics",
		echo.WrapHandler(promhttp.Handler()),
		echomw.BasicAuth(bav),
	)
}
