// Package handler は、ルーティングの登録と管理を行うための機能を提供します。
package handler

import (
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// RouteMount は、サーバーのルーティングを登録するための関数型です。
type RouteMount func(*echo.Echo)

// RouteMountIn は、ルーティングの登録に使用される入力構造体です。
type RouteMountIn struct {
	fx.In
	Registrars []RouteMount `group:"routes"`
}

// RouteMountOut は、ルーティングの登録に使用される出力構造体です。
type RouteMountOut struct {
	fx.Out
	Registrar RouteMount `group:"routes"`
}

// RouteMounted は、ルートが登録されたことを示すトークンです。
type RouteMounted struct{}

// MountRoutes は、すべてのルートレジストラをサーバーインスタンスに登録します。
func MountRoutes(
	e *echo.Echo, in RouteMountIn,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *httpstack.AppliedServerExtends,
) *RouteMounted {
	for _, r := range in.Registrars {
		r(e)
	}
	return &RouteMounted{}
}
