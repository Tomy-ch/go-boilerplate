// Package timeout は、リクエスト全体の deadline budget を設定するミドルウェアを提供します。
package timeout

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Middleware は、リクエスト context に timeout の deadline を設定するミドルウェアを返します。
//
// per-request deadline を1点設定し、後続の全ミドルウェア・ハンドラ・DB クエリ・外部 HTTP が
// 単一の budget を ctx 経由で共有します。response writer のデータ競合を避けるため、
// echo 標準の race-free な ContextTimeout を基底とします（deprecated な Timeout は競合を抱える）。
//
// deadline 超過時は apperror.ErrUnavailable を返し、他のエラーと同じボディ形（HTTP 503）を維持します。
// deadline 超過以外のエラー（ハンドラが返した通常のアプリエラー等）はそのまま伝播し、本来の
// ステータスマッピングを維持します。
func Middleware(timeout time.Duration) echo.MiddlewareFunc {
	return middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: timeout,
		ErrorHandler: func(err error, _ echo.Context) error {
			if errors.Is(err, context.DeadlineExceeded) {
				return xerrors.Wrap(apperror.ErrUnavailable, "request deadline exceeded")
			}
			return err
		},
	})
}
