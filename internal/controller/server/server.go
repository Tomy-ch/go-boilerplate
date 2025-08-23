// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"context"
	"strconv"
	"time"

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
	lc fx.Lifecycle, e *echo.Echo, cfg *config.Config, z *zap.Logger,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *httpstack.AppliedServerExtends,
) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			addr := cfg.ServerPort()
			go func() {
				if err := e.Start(":" + strconv.Itoa(addr)); err != nil {
					z.Error("failed to start http server", zap.Error(err))
				}
			}()
			z.Info("http started",
				zap.String("port", strconv.Itoa(addr)),
				zap.Strings("allowed origins", cfg.AllowedOrigins()),
				zap.String("cidr", cfg.CIDR().IP.String()),
				zap.String("mode", cfg.ServerAppMode()),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownTime := cfg.ServerShutdownTimeout()
			ctx, cancel := context.WithTimeout(ctx, shutdownTime*time.Second)
			defer cancel()
			z.Info("http stopping")
			return e.Shutdown(ctx)
		},
	})
}
