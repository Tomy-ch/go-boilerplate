// Package echorecover は、パニックからのリカバリを行うミドルウェアです。
package echorecover

import (
	"boilerplate-go/internal/appconfig"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
)

const (
	productionStackSize  = 4 << 10
	developmentStackSize = 10 << 10
)

// Middleware は、Echoフレームワークのミドルウェアで、パニックからのリカバリを行います。
func Middleware(logger *zap.Logger, cfg *appconfig.Config) echo.MiddlewareFunc {
	var conf middleware.RecoverConfig
	switch {
	case cfg.IsAppDevelopmentMode():
		conf = developmentConfig()
	case cfg.IsAppProductionMode():
		conf = productionConfig()
	default:
		conf = productionConfig()
		logger.Warn(
			"Unknown environment, using production config for recover middleware",
			zap.String("env", cfg.AppMode()),
		)
	}

	// ログ出力の設定
	conf.LogErrorFunc = func(c echo.Context, err error, stack []byte) error {
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

	return middleware.RecoverWithConfig(conf)
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
