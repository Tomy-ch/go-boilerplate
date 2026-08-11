package redmetrics

import (
	"sync"
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"
	"go-boilerplate/internal/controller/server"

	"github.com/labstack/echo/v5"
)

// Middleware は、HTTP リクエストの count / duration / status を計測する Echo ミドルウェアを返します。
// Observe は 1 リクエストにつき厳密に 1 回だけ呼び出されます。/metrics などの運用系パスは計測対象から
// 除外し、レスポンスを取り出せない場合は計測せず素通しします。After フック方式に起因する既知の限界
// （ボディ無し応答が計測されない等）は README を参照してください。
func Middleware(rec Recorder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if ops.IsOpsPath(c.Request().URL.Path) {
				return next(c)
			}

			res := server.ResponseOf(c)
			if res == nil {
				return next(c)
			}

			start := time.Now()
			method := c.Request().Method

			var once sync.Once
			res.After(func() {
				once.Do(func() {
					status := normalizeStatus(res.Status)
					rec.Observe(method, routeOf(c), status, statusClass(status), time.Since(start))
				})
			})

			return next(c)
		}
	}
}
