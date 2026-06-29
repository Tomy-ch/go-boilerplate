package redmetrics

import (
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"

	"github.com/labstack/echo/v4"
)

// Middleware は、HTTP リクエストの count / duration / status を計測する Echo ミドルウェアを返します。
// status は c.Response().After フックで確定後に取得し、error handler / recovery による
// 最終 status を安全に観測します。/metrics などの運用系パスは計測対象から除外します。
func Middleware(rec Recorder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if ops.IsOpsPath(c.Request().URL.Path) {
				return next(c)
			}

			start := time.Now()
			method := c.Request().Method

			c.Response().After(func() {
				status := normalizeStatus(c.Response().Status)
				rec.Observe(method, routeOf(c), status, statusClass(status), time.Since(start))
			})

			return next(c)
		}
	}
}
