package observability

import (
	"github.com/exaring/otelpgx"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
)

// NewPgxTracer は、TracerProvider / MeterProvider を明示注入した otelpgx トレーサーを生成します。
//
// provider を引数で受けることで、グローバル provider の登録順に依存せず span / metric を送出します。
// 接続情報（DB ユーザー名等）は span 属性に含めません。
func NewPgxTracer(tp trace.TracerProvider, mp *sdkmetric.MeterProvider) *otelpgx.Tracer {
	return otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tp),
		otelpgx.WithMeterProvider(mp),
		otelpgx.WithDisableConnectionDetailsInAttributes(),
	)
}
