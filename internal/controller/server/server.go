// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"boilerplate-go/internal/appconfig"

	"github.com/labstack/echo/v4"
)

func New(
	cfg *appconfig.Config,
	validator echo.Validator,
	binder echo.Binder,
	ipextractor echo.IPExtractor,
) *echo.Echo {
	e := echo.New()

	setPrimitiveEchoSettings(e, cfg)
	setCustomEchoBindings(e, validator, binder, ipextractor)

	return e
}

// setPrimitiveEchoSettings は、Echoの基本的なプロパティを設定します。
func setPrimitiveEchoSettings(
	e *echo.Echo,
	cfg *appconfig.Config,
) {
	isProduction := cfg.IsAppProductionMode()
	isDevelopment := cfg.IsAppDevelopmentMode()

	// 開発モード向けのデバッグ支援機能(詳細なエラーメッセージ表示)
	e.Debug = isDevelopment
	// バナーは本番環境では非表示にする
	e.HideBanner = isProduction
	// ポート番号は本番環境では非表示にする
	e.HidePort = isProduction
}

// setCustomEchoBindings は、Echoにカスタムバインディングを設定します。
func setCustomEchoBindings(
	e *echo.Echo,
	validator echo.Validator,
	binder echo.Binder,
	ipextractor echo.IPExtractor,
) {
	e.Validator = validator
	e.Binder = binder
	e.IPExtractor = ipextractor
}
