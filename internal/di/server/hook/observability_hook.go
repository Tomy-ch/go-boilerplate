package hook

import (
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/observability"
)

// RegisterObservabilityShutdownHooks は、アプリケーションのシャットダウン時に
// TracerProvider / MeterProvider を Shutdown するためのフックを登録します。
func RegisterObservabilityShutdownHooks(reg lifecycle.Registrar, shutdowner observability.ProviderShutdowner) {
	reg.RegisterStop(shutdowner.Shutdown)
}
