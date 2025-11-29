// Package httpstack は、サーバー機能の拡張設定を提供します。
package httpstack

import (
	"sort"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServerExtends は、サーバーの拡張機能を表します。
type ServerExtends struct {
	fx.In
	// PreList は、Preミドルウェアとして適用されるミドルウェアのリストです。
	PreList []echo.MiddlewareFunc `group:"middlewares.pre"`
	// UseList は、Useミドルウェアとして適用されるミドルウェアのリストです。
	UseList []UseMiddleware `group:"middlewares.use"`
	// CfgList は、サーバーに副作用で適用される設定関数のリストです。
	CfgList []SrvCfg `group:"server.configurators"`
}

// AppliedServerExtends は、サーバー拡張が適用されたことを示すトークンです。
type AppliedServerExtends struct{}

// SrvCfg は、サーバーの設定関数を表します。
type SrvCfg func(*echo.Echo)

// ServeCfgOut は、サーバーの設定の出力時に使用される構造体です。
type ServeCfgOut struct {
	fx.Out
	SrvCfg SrvCfg `group:"server.configurators"`
}

// UseMiddleware は、Use ミドルウェアとその適用順序を表します。
type UseMiddleware struct {
	// Priority は、ミドルウェアの適用順序を表します（小さい方が先に適用される）
	Priority int
	// Middleware は、適用対象の Echo ミドルウェアです。
	Middleware echo.MiddlewareFunc
}

// UseMiddlewareOut は、fx の group 出力用のラッパーです。
type UseMiddlewareOut struct {
	fx.Out
	Middleware UseMiddleware `group:"middlewares.use"`
}

// ApplyExtends は、サーバー拡張を適用します。
func ApplyExtends(e *echo.Echo, logger *zap.Logger, extends ServerExtends) *AppliedServerExtends {
	ApplyPreMiddlewares(e, logger, extends.PreList)
	ApplyUseMiddlewares(e, logger, extends.UseList)
	ApplyConfigurators(e, logger, extends.CfgList)
	return &AppliedServerExtends{}
}

// ApplyPreMiddlewares は、Echoに対してPreのミドルウェアを適用します。
func ApplyPreMiddlewares(e *echo.Echo, logger *zap.Logger, mws []echo.MiddlewareFunc) {
	logger.Info("Applying pre middleware", zap.Int("count", len(mws)))
	for _, mw := range mws {
		e.Pre(mw)
	}
}

// ApplyUseMiddlewares は、Echoに対してUseのミドルウェアを適用します。
func ApplyUseMiddlewares(e *echo.Echo, logger *zap.Logger, mws []UseMiddleware) {
	logger.Info("Applying use middleware", zap.Int("count", len(mws)))

	// Order でソートしてから適用
	sort.Slice(mws, func(i, j int) bool {
		return mws[i].Priority < mws[j].Priority
	})

	for _, mw := range mws {
		e.Use(mw.Middleware)
	}
}

// ApplyConfigurators は、Echoに対して設定関数を適用します。
func ApplyConfigurators(e *echo.Echo, logger *zap.Logger, cfgs []SrvCfg) {
	logger.Info("Applying server configurator", zap.Int("count", len(cfgs)))
	for _, cfg := range cfgs {
		cfg(e)
	}
}
