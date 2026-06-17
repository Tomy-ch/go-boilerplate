//go:build otelverify

package observability

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/system"

	"github.com/stretchr/testify/require"
)

// captureReg は、検証中に登録された Stop フックを保持する簡易 Registrar。
type captureReg struct {
	stops []func(context.Context) error
}

func (r *captureReg) RegisterStart(func(context.Context) error) {}

func (r *captureReg) RegisterStop(f func(context.Context) error) {
	r.stops = append(r.stops, f)
}

// TestExportVerify は、実装した provider が OTLP で実際にエクスポートし受信側で取得できるかを確認する
// 手動 e2e ハーネス。ビルドタグ otelverify で隔離しており、通常の test / lint / カバレッジには含まれない。
//
// 実行手順:
//  1. 受信側 Collector を起動する（debug exporter が受信内容を標準出力にダンプする）:
//     docker run --rm -p 4317:4317 -p 4318:4318 \
//       -v "$PWD/internal/observability/testdata/otel-collector.yaml":/cfg.yaml \
//       otel/opentelemetry-collector-contrib --config=/cfg.yaml
//  2. 別ターミナルで本テストを実行する:
//     go test -tags otelverify -run TestExportVerify -count=1 ./internal/observability/
//  3. Collector のログに、service.name / service.version / deployment.environment.name つきの
//     span（otel-verify-span）と、go.* のランタイムメトリクスが出力されることを確認する。
func TestExportVerify(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
	res, err := NewResource(appCfg, system.NewBuildInfo())
	require.NoError(t, err)

	reg := &captureReg{}

	tp, err := TracerProvider(reg, res)
	require.NoError(t, err)

	_, err = MeterProvider(reg, res)
	require.NoError(t, err)

	tr := tp.Tracer("otel-verify")
	_, span := tr.Start(context.Background(), "otel-verify-span")
	span.End()

	// 登録された Stop フック(tp.Shutdown / mp.Shutdown)を実行し、span とメトリクスをフラッシュする。
	for _, stop := range reg.stops {
		require.NoError(t, stop(context.Background()))
	}
}
