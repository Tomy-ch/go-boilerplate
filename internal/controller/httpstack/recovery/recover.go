// Package recovery は、パニックからのリカバリを行うミドルウェアです。
package recovery

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const (
	// productionStackSize は、本番環境でのスタックサイズです。
	productionStackSize = 4 << 10 // 4KB
	// developmentStackSize は、開発環境でのスタックサイズです。
	developmentStackSize = 10 << 10 // 10KB
)

// Middleware は、Echoフレームワークのミドルウェアで、パニックからのリカバリを行います。
//
// [middleware.RecoverWithConfig] は復帰した panic を [middleware.PanicStackError] へ包んで
// 戻り値のエラーとして返すため、その外側でスタック付きのログを出し、
// エラーハンドラへは包む前のエラーを渡します。
func Middleware(z logging.Logger, lf logging.LogFieldBuilder, appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	recoverMW := middleware.RecoverWithConfig(newRecoverConfig(z, appCfg))
	logPanic := newPanicLogFunc(z, lf)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		recovered := recoverMW(next)
		return func(c *echo.Context) error {
			err := recovered(c)

			var pse *middleware.PanicStackError
			if !xerrors.As(err, &pse) {
				return err
			}
			logPanic(c, pse.Err, pse.Stack)
			return pse.Err
		}
	}
}

// newRecoverConfig は、環境設定に基づいてリカバリミドルウェアの設定を生成します。
func newRecoverConfig(logger logging.Logger, appCfg *config.ApplicationConfig) middleware.RecoverConfig {
	switch {
	case appCfg.IsDevelopmentMode():
		return developmentConfig()
	case appCfg.IsProductionMode():
		return productionConfig()
	default:
		logger.Named("middleware.recover").Warn(
			context.Background(),
			"Unknown environment, using production config for recover middleware",
			logging.String("env", appCfg.Mode()),
		)
		return productionConfig()
	}
}

// newPanicLogFunc は、復帰したパニックのログ出力関数を生成します。
func newPanicLogFunc(logger logging.Logger, lf logging.LogFieldBuilder) func(c *echo.Context, err error, stack []byte) {
	return func(c *echo.Context, err error, stack []byte) {
		reqIn := server.BuildHTTPRequestLogInput(c, logging.EventTypePanic)
		recoverFields := []*logging.Field{
			logging.String(logging.InternalErrorKey, err.Error()),
			logging.Strings(logging.InternalStackTraceKey, logging.SplitStackLines(string(stack))),
		}
		fields := append(lf.BuildHTTPRequestFields(reqIn), recoverFields...)
		logger.Named("middleware.recover").Error(c.Request().Context(), "panic recovered", fields...)
		// ログ済みを記録する（二重ログは ctxhelper.GetRecoveredFromEcho で抑止）。
		ctxhelper.SetRecoveredToEcho(c, true)
	}
}

// developmentConfig は、開発環境用のリカバリミドルウェアの設定を返します。
func developmentConfig() middleware.RecoverConfig {
	return middleware.RecoverConfig{
		StackSize:         developmentStackSize,
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
}

// productionConfig は、本番環境用のリカバリミドルウェアの設定を返します。
func productionConfig() middleware.RecoverConfig {
	return middleware.RecoverConfig{
		StackSize: productionStackSize,
		// DisableStackAll=true で他 goroutine は除外しつつ、当該 goroutine のスタックは捕捉する
		// （DisablePrintStack=true だと echo が runtime.Stack 自体を行わず PanicStackError にも包まれない）。
		DisableStackAll:   true,
		DisablePrintStack: false,
	}
}
