package observability

import (
	"context"

	"go-boilerplate/internal/di/lifecycle"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider は、OpenTelemetry のトレーサープロバイダーを初期化し、otel.SetTracerProvider で
// グローバル登録したうえで、シャットダウン時に Shutdown を呼ぶフックを Registrar へ登録して返します。
// Exporter / SpanProcessor は未配線（最小構成）のため、span を実際に送出するには利用側で
// WithBatcher 等を追加してください。
func TracerProvider(reg lifecycle.Registrar) trace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.Default()),
	)

	otel.SetTracerProvider(tp)

	reg.RegisterStop(func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	return tp
}
