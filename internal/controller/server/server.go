// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"context"
	"strconv"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// New は、サーバーインスタンスを作成します。
func New() *echo.Echo {
	return echo.New()
}

// ServeHTTP は、HTTPサーバーを起動します。
func ServeHTTP(
	lc fx.Lifecycle, e *echo.Echo, z logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *httpstack.AppliedServerExtends,
) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			addr := srvCfg.Port()
			go func() {
				if err := e.Start(":" + strconv.Itoa(addr)); err != nil {
					z.Named("server.Start").Error("failed to start http server", logging.Error("e.Start", err))
				}
			}()
			z.Named("server.Start").Info("http started",
				logging.String("port", strconv.Itoa(addr)),
				logging.Strings("allowed_origins", secCfg.AllowedOrigins()),
				logging.String("cidr", secCfg.CIDR().IP.String()),
				logging.String("mode", appCfg.Mode()),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownTime := srvCfg.ShutdownTimeout()
			ctx, cancel := context.WithTimeout(ctx, shutdownTime)
			defer cancel()
			z.Info("http stopping")
			return e.Shutdown(ctx)
		},
	})
}
