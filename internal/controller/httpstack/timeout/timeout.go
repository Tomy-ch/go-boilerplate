// Package timeout は、リクエスト全体の deadline budget を設定するミドルウェアを提供します。
package timeout

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Middleware は、リクエスト context に timeout の deadline を設定するミドルウェアを返します。
//
// echo 標準の middleware.ContextTimeout を薄くラップします（自前実装は再発明かつ
// response writer のデータ競合 risk を抱えるため、race-free な ContextTimeout を用いる）。
// deadline を request context に載せることで、後続の全ミドルウェア・ハンドラ・DB クエリ・
// 外部 HTTP が単一の budget を ctx 経由で共有します（M1 deadline budget）。
//
// deadline 超過時は apperror.ErrUnavailable に wrap して返し、echo 中央の統一
// HTTPErrorHandler に委譲することで、エラーボディ形を他のエラーと揃えます。
func Middleware(timeout time.Duration) echo.MiddlewareFunc {
	return middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Timeout: timeout,
		ErrorHandler: func(_ error, _ echo.Context) error {
			return xerrors.Wrap(apperror.ErrUnavailable, "request deadline exceeded")
		},
	})
}
