package redmetrics

import (
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"

	"github.com/labstack/echo/v4"
)

// Middleware は、HTTP リクエストの count / duration / status を計測する Echo ミドルウェアを返します。
// status は c.Response().After フックで確定後に取得し、error handler / recovery による
// 最終 status を安全に観測します。/metrics などの運用系パスは計測対象から除外します。
//
// 計測の既知の限界（After フック方式に起因する仕様）:
//   - ボディ無し応答（204 No Content / 304 Not Modified など）は Write が呼ばれず
//     After フックが発火しないため、計測されない。
//   - ストリーミング応答のように Write が複数回呼ばれる場合、After フックも複数回発火し、
//     同一リクエストが重複計上されうる。
//
// これらは status を確定後に安全に観測するための After フック採用とのトレードオフであり、
// 現時点では仕様として許容する。
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
