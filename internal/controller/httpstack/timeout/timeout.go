// Package timeout は、リクエスト全体の deadline budget を設定するミドルウェアを提供します。
package timeout

import (
	"context"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Middleware は、リクエスト context に timeout の deadline を設定するミドルウェアを返します。
// deadline 超過は apperror.ErrUnavailable(503) へ、それ以外のエラーはそのまま伝播します。
// 単一 deadline budget の共有と ContextTimeout を基底に選んだ理由は README を参照してください。
func Middleware(timeout time.Duration) echo.MiddlewareFunc {
	return middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: timeout,
		ErrorHandler: func(_ *echo.Context, err error) error {
			if xerrors.Is(err, context.DeadlineExceeded) {
				return xerrors.Join(apperror.ErrUnavailable, err)
			}
			return err
		},
	})
}
