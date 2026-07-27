// Package timeout は、リクエスト全体の deadline budget を設定するミドルウェアを提供します。
package timeout

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Middleware は、リクエスト context に timeout の deadline を設定するミドルウェアを返します。
//
// 後続の全ミドルウェア・ハンドラ・DB・外部 HTTP が単一の deadline budget を ctx 経由で共有します。
// response writer のデータ競合を避けるため race-free な ContextTimeout を基底とします。
// deadline 超過は apperror.ErrUnavailable(503) へ、それ以外のエラーはそのまま伝播します。
func Middleware(timeout time.Duration) echo.MiddlewareFunc {
	return middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: timeout,
		ErrorHandler: func(err error, _ echo.Context) error {
			if xerrors.Is(err, context.DeadlineExceeded) {
				// 原因 err を保持したまま結合し、ログでの原因追跡を保つ。
				return xerrors.Join(apperror.ErrUnavailable, err)
			}
			return err
		},
	})
}
