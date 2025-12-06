// Package server は、サーバーインスタンスを初期化します。
package server

import (
	"context"
	"strconv"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/di/lifecycle"
	"boilerplate-go/internal/di/server/extension"
	"boilerplate-go/internal/logging"

	"github.com/labstack/echo/v4"
)

// NewAppServer は、サーバーインスタンスを作成します。
func NewAppServer() *echo.Echo {
	return echo.New()
}

// ServeHTTP は、HTTPサーバーを起動します。
func ServeHTTP(
	e *echo.Echo,
	reg lifecycle.Registrar,
	log logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *extension.AppliedServerExtends,
) {
	reg.RegisterStart(newStartServerFunc(e, srvCfg, log, secCfg, appCfg))
	reg.RegisterStop(newStopServerFunc(e, log, srvCfg))
}

// newStartServerFunc は、HTTPサーバーを起動する関数を生成します。
func newStartServerFunc(
	e *echo.Echo,
	srvCfg *config.ServerConfig,
	log logging.Logger,
	secCfg *config.SecurityConfig,
	appCfg *config.ApplicationConfig,
) func(context.Context) error {
	return func(_ context.Context) error {
		addr := srvCfg.Port()
		go func() {
			if err := e.Start(":" + strconv.Itoa(addr)); err != nil {
				log.Named("server.Start").Error("failed to start http server", logging.Error("e.Start", err))
			}
		}()
		log.Named("server.Start").Info("http started",
			logging.String("port", strconv.Itoa(addr)),
			logging.Strings("allowed_origins", secCfg.AllowedOrigins()),
			logging.String("cidr", secCfg.CIDR().IP.String()),
			logging.String("mode", appCfg.Mode()),
		)
		return nil
	}
}

// newStopServerFunc は、HTTPサーバーを停止する関数を生成します。
func newStopServerFunc(
	e *echo.Echo, log logging.Logger, srvCfg *config.ServerConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		shutdownTime := srvCfg.ShutdownTimeout()
		ctx, cancel := context.WithTimeout(ctx, shutdownTime)
		defer cancel()
		log.Info("http stopping")
		return e.Shutdown(ctx)
	}
}
