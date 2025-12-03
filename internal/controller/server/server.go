// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"context"
	"strconv"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// New は、サーバーインスタンスを作成します。
func New() *echo.Echo {
	return echo.New()
}

// ServeHTTP は、HTTPサーバーを起動します。
func ServeHTTP(
	lc fx.Lifecycle, e *echo.Echo, z *zap.Logger,
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
					z.Error("failed to start http server", zap.Error(err))
				}
			}()
			z.Info("http started",
				zap.String("port", strconv.Itoa(addr)),
				zap.Strings("allowed_origins", secCfg.AllowedOrigins()),
				zap.String("cidr", secCfg.CIDR().IP.String()),
				zap.String("mode", appCfg.Mode()),
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
