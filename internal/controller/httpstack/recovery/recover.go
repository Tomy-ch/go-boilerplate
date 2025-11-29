// Package recovery は、パニックからのリカバリを行うミドルウェアです。
package recovery

import (
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
)

const (
	// productionStackSize は、本番環境でのスタックサイズです。
	productionStackSize = 4 << 10 // 4KB
	// developmentStackSize は、開発環境でのスタックサイズです。
	developmentStackSize = 10 << 10 // 10KB
)

// Middleware は、Echoフレームワークのミドルウェアで、パニックからのリカバリを行います。
func Middleware(logger *zap.Logger, appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	cnf := newRecoverConfig(logger, appCfg)
	cnf.LogErrorFunc = newRecoverLogErrorFunc(logger)

	return middleware.RecoverWithConfig(cnf)
}

// newRecoverConfig は、環境設定に基づいてリカバリミドルウェアの設定を生成します。
func newRecoverConfig(logger *zap.Logger, appCfg *config.ApplicationConfig) middleware.RecoverConfig {
	switch {
	case appCfg.IsAppDevelopmentMode():
		return developmentConfig()
	case appCfg.IsAppProductionMode():
		return productionConfig()
	default:
		logger.Warn(
			"Unknown environment, using production config for recover middleware",
			zap.String("env", appCfg.AppMode()),
		)
		return productionConfig()
	}
}

// newRecoverLogErrorFunc は、リカバリミドルウェアのログ出力関数を生成します。
func newRecoverLogErrorFunc(logger *zap.Logger) func(c echo.Context, err error, stack []byte) error {
	return func(c echo.Context, err error, stack []byte) error {
		req := c.Request()
		logger.Error("panic recovered",
			zap.Error(err),
			zap.ByteString("stack", stack),
			zap.String("method", req.Method),
			zap.String("uri", req.RequestURI),
			zap.String("remote_ip", c.RealIP()),
		)
		return nil
	}
}

// developmentConfig は、開発環境用のリカバリミドルウェアの設定を返します。
func developmentConfig() middleware.RecoverConfig {
	return middleware.RecoverConfig{
		StackSize:         developmentStackSize,
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogLevel:          log.DEBUG,
	}
}

// productionConfig は、本番環境用のリカバリミドルウェアの設定を返します。
func productionConfig() middleware.RecoverConfig {
	return middleware.RecoverConfig{
		StackSize:         productionStackSize,
		DisableStackAll:   true,
		DisablePrintStack: true,
		LogLevel:          log.ERROR,
	}
}
