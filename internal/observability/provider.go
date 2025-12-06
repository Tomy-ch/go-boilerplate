//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

package observability

import (
	"context"

	"boilerplate-go/internal/di/lifecycle"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider は、OpenTelemetryのトレーサープロバイダーを初期化して、
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
