package redmetrics

import (
	"sync"
	"time"

	"go-boilerplate/internal/controller/httpstack/ops"

	"github.com/labstack/echo/v4"
)

// Middleware は、HTTP リクエストの count / duration / status を計測する Echo ミドルウェアを返します。
// status は c.Response().After フックで確定後に取得し、error handler / recovery による
// 最終 status を安全に観測します。/metrics などの運用系パスは計測対象から除外します。
//
// After フックはレスポンスの Write ごとに発火するため、ストリーミング応答（チャンク / SSE 等）
// では複数回呼ばれうる。これによる重複計上を防ぐため、リクエストごとに確保した sync.Once で
// フック本体をガードし、Observe を 1 リクエストにつき厳密に 1 回だけ呼び出す。
//
// 計測の既知の限界（After フック方式に起因する仕様）:
//   - ボディ無し応答（204 No Content / 304 Not Modified など）は Write が呼ばれず
//     After フックが発火しないため、計測されない。
//
// これは status を確定後に安全に観測するための After フック採用とのトレードオフであり、
// 現時点では仕様として許容する。
func Middleware(rec Recorder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if ops.IsOpsPath(c.Request().URL.Path) {
				return next(c)
			}

			start := time.Now()
			method := c.Request().Method

			var once sync.Once
			c.Response().After(func() {
				once.Do(func() {
					status := normalizeStatus(c.Response().Status)
					rec.Observe(method, routeOf(c), status, statusClass(status), time.Since(start))
				})
			})

			return next(c)
		}
	}
}
