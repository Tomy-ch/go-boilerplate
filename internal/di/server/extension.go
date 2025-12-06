package server

import (
	"boilerplate-go/internal/di/server/extension"
	"boilerplate-go/internal/di/server/extension/decoration"
	"boilerplate-go/internal/di/server/extension/inbound"
	"boilerplate-go/internal/di/server/extension/instrumentation"
	"boilerplate-go/internal/di/server/extension/nonprod"
	"boilerplate-go/internal/di/server/extension/outbound"
	"boilerplate-go/internal/di/server/extension/security"

	"go.uber.org/fx"
)

// HTTPStackModule は、HTTP スタック関連の依存関係を提供するfx.Moduleです。
func HTTPStackModule() fx.Option {
	return fx.Module("httpstack",
		// Middleware Modules
		decoration.BannerModule(),
		decoration.DefaultPortModule(),
		decoration.DefaultPortModule(),
		inbound.IPExtractorModule(),
		inbound.URIModule(),
		inbound.ValidatorModule(),
		outbound.ErrorHandlerModule(),
		outbound.ForceJSONModule(),
		outbound.RecoveryModule(),
		security.Module(),
		security.CORSModule(),
		security.CookieModule(),
		instrumentation.RequestIDModule(),
		instrumentation.LoggingModule(),
		instrumentation.ObservabilityModule(),
		nonprod.DebugModeModule(),
		fx.Provide(
			extension.ApplyExtends,
		),
	)
}
