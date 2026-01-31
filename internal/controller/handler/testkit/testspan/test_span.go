// Package testspan は、テスト用のトレーススパンユーティリティを提供します。
package testspan

import (
	"testing"

	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
)

// StartTestSpanForEcho は、echo.Context の Request に tracer.Start で開始した
// context を埋め込み、終了用の endSpan 関数を返します。
func StartTestSpanForEcho(
	t *testing.T,
	c echo.Context,
) (echo.Context, func()) {
	t.Helper()

	req := c.Request()

	ctx, span := observability.NewStubSpanContext(t)
	req = req.WithContext(ctx)
	c.SetRequest(req)

	return c, span
}
