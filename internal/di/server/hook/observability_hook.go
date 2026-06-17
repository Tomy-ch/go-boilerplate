package hook

import (
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/observability"
)

// RegisterObservabilityShutdownHooks は、アプリケーションのシャットダウン時に
// TracerProvider / MeterProvider を Shutdown するためのフックを登録します。
// 構築（observability.NewTracerProvider / NewMeterProvider）はライフサイクル非依存で行われ、
// その後始末（Shutdown）の登録を di 層であるこの hook が担います。
// otel SDK 型を di 層へ漏らさないため、具象ではなく observability.ProviderShutdowner に依存します。
func RegisterObservabilityShutdownHooks(reg lifecycle.Registrar, shutdowner observability.ProviderShutdowner) {
	reg.RegisterStop(shutdowner.Shutdown)
}
