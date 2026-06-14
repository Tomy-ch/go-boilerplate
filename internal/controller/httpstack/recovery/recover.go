// Package recovery は、パニックからのリカバリを行うミドルウェアです。
package recovery

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// productionStackSize は、本番環境でのスタックサイズです。
	productionStackSize = 4 << 10 // 4KB
	// developmentStackSize は、開発環境でのスタックサイズです。
	developmentStackSize = 10 << 10 // 10KB
)

// Middleware は、Echoフレームワークのミドルウェアで、パニックからのリカバリを行います。
func Middleware(z logging.Logger, lf logging.LogFieldBuilder, appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	cnf := newRecoverConfig(z, appCfg)
	cnf.LogErrorFunc = newRecoverLogErrorFunc(z, lf)

	return middleware.RecoverWithConfig(cnf)
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
			"Unknown environment, using production config for recover middleware",
			logging.String("env", appCfg.Mode()),
		)
		return productionConfig()
	}
}

// newRecoverLogErrorFunc は、リカバリミドルウェアのログ出力関数を生成します。
func newRecoverLogErrorFunc(logger logging.Logger, lf logging.LogFieldBuilder) func(c echo.Context, err error, stack []byte) error {
	return func(c echo.Context, err error, stack []byte) error {
		reqIn := server.BuildHTTPRequestLogInput(c, logging.EventTypePanic)
		recoverFields := []*logging.Field{
			logging.String(logging.InternalErrorKey, err.Error()),
			logging.Strings(logging.InternalStackTraceKey, logging.SplitStackLines(string(stack))),
		}
		fields := append(lf.BuildHTTPRequestFields(reqIn), recoverFields...)
		logger.Named("middleware.recover").Error("panic recovered", fields...)
		// ログ済みを記録し err を返す（echo が c.Error で 500 を返す。二重ログは IsRecovered で抑止）。
		server.MarkRecovered(c)
		return err
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
		// （DisablePrintStack=true だと echo が runtime.Stack 自体を行わず LogErrorFunc に空が渡る）。
		DisableStackAll:   true,
		DisablePrintStack: false,
	}
}
