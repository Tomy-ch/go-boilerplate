// Package testspan は、テスト用のトレーススパンユーティリティを提供します。
package testspan

import (
	"testing"

	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v5"
)

// StartTestSpanForEcho は、テスト用のスタブトレーススパンを *echo.Context に設定し、スパン終了関数を返します。
func StartTestSpanForEcho(
	t *testing.T,
	c *echo.Context,
) (*echo.Context, func()) {
	t.Helper()

	req := c.Request()

	ctx, endSpan := observability.NewStubSpanContext(t)
	req = req.WithContext(ctx)
	c.SetRequest(req)

	return c, endSpan
}
