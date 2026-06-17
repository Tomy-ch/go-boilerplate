//go:generate mockgen -source=$GOFILE -destination=mock/mock_shutdown.gen.go -package=mock_$GOPACKAGE

package observability

import (
	"context"
	"errors"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ProviderShutdowner は TracerProvider / MeterProvider の Shutdown をまとめた後始末ハンドル。
type ProviderShutdowner interface {
	// Shutdown は TracerProvider / MeterProvider を Shutdown し、発生したエラーを結合して返す。
	Shutdown(ctx context.Context) error
}

type providerShutdowner struct {
	tp *sdktrace.TracerProvider
	mp *sdkmetric.MeterProvider
}

// NewProviderShutdowner は具象プロバイダから後始末ハンドルを構築する。
func NewProviderShutdowner(tp *sdktrace.TracerProvider, mp *sdkmetric.MeterProvider) ProviderShutdowner {
	return providerShutdowner{tp: tp, mp: mp}
}

func (s providerShutdowner) Shutdown(ctx context.Context) error {
	return errors.Join(s.tp.Shutdown(ctx), s.mp.Shutdown(ctx))
}
