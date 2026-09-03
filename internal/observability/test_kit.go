package observability

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	tracer           = "noop-tracer"
	layer  layerName = "noop-layer"
	pkg              = "noop-pkg"
	span             = "noop-span"
)

// ObservedHTTPClientMetrics は、計上内容を読み出せる HTTPClientMetrics です。
type ObservedHTTPClientMetrics struct {
	*HTTPClientMetrics

	reader *sdkmetric.ManualReader
}

// ObservedRealtimeMetrics は、計上内容（label と値）を読み出せる RealtimeMetrics です。
type ObservedRealtimeMetrics struct {
	*RealtimeMetrics

	reader *sdkmetric.ManualReader
}

// spanRecorder は、終了した span をそのまま保持する同期 exporter です。
type spanRecorder struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

// NewNoopTracerFactory は、テスト用に TracerFactory を無効化して返します。
func NewNoopTracerFactory(t *testing.T) TracerFactory {
	t.Helper()

	tp := noop.NewTracerProvider()
	return NewTracerFactory(tp)
}

// NewMockControllerLayerTracer は、テスト用のコントローラーレイヤートレーサーを生成します。
func NewMockControllerLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Controller()
}

// NewMockUsecaseLayerTracer は、テスト用のユースケースレイヤートレーサーを生成します。
func NewMockUsecaseLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Usecase()
}

// NewMockInfraLayerTracer は、テスト用のインフラレイヤートレーサーを生成します。
func NewMockInfraLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Infra()
}

// NewNoopLayerTracer は、テスト用に LayerTracer を無効化して返します。
func NewNoopLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	return LayerTracer{
		tracer:  noop.NewTracerProvider().Tracer(tracer),
		layer:   layer,
		pkgName: pkg,
	}
}

// NewNoopWorkerMetrics は、テスト用に no-op の MeterProvider から WorkerMetrics を生成します。
func NewNoopWorkerMetrics(t *testing.T) *WorkerMetrics {
	t.Helper()
	wm, err := NewWorkerMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop worker metrics: %v", err)
	}
	return wm
}

// NewNoopHTTPClientMetrics は、テスト用に no-op の MeterProvider から HTTPClientMetrics を生成します。
func NewNoopHTTPClientMetrics(t *testing.T) *HTTPClientMetrics {
	t.Helper()
	hm, err := NewHTTPClientMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop http client metrics: %v", err)
	}
	return hm
}

// NewObservedRealtimeMetrics は、計上内容を読み出せる RealtimeMetrics を返します。
// 計上先の label と値を検証したい呼び出し元テストで使用します。
func NewObservedRealtimeMetrics(t *testing.T) *ObservedRealtimeMetrics {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	rm, err := NewRealtimeMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatalf("failed to build observed realtime metrics: %v", err)
	}

	return &ObservedRealtimeMetrics{RealtimeMetrics: rm, reader: reader}
}

// CounterValue は、metricName の counter のうち labelKey=labelValue の点の値を返します。
// 該当する点が無ければ -1 を返します（0 は「0 が計上された」であり、両者は区別すべき別の事実です）。
func (o *ObservedRealtimeMetrics) CounterValue(t *testing.T, metricName, labelKey, labelValue string) int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := o.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect realtime metrics: %v", err)
	}

	m, found := findMetric(rm, metricName)
	if !found {
		return -1
	}

	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("metric %s is not an int64 counter", metricName)
	}

	for _, dp := range sum.DataPoints {
		if matchesLabel(dp.Attributes, labelKey, labelValue) {
			return dp.Value
		}
	}

	return -1
}

// findMetric は、収集結果から名前が一致する metric を 1 件返します。
func findMetric(rm metricdata.ResourceMetrics, metricName string) (metricdata.Metrics, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName {
				return m, true
			}
		}
	}

	return metricdata.Metrics{}, false
}

// matchesLabel は、データポイントが labelKey=labelValue を持つかを返します。
func matchesLabel(attrs attribute.Set, labelKey, labelValue string) bool {
	v, found := attrs.Value(attribute.Key(labelKey))

	return found && v.AsString() == labelValue
}

// HistogramCount は、metricName の histogram に記録された点の数を返します（未記録なら 0）。
func (o *ObservedRealtimeMetrics) HistogramCount(t *testing.T, metricName string) uint64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := o.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect realtime metrics: %v", err)
	}

	m, found := findMetric(rm, metricName)
	if !found {
		return 0
	}

	switch h := m.Data.(type) {
	case metricdata.Histogram[float64]:
		return totalHistogramCount(h.DataPoints)
	case metricdata.Histogram[int64]:
		return totalHistogramCount(h.DataPoints)
	default:
		t.Fatalf("metric %s is not a histogram", metricName)

		return 0
	}
}

// totalHistogramCount は、全データポイントの記録回数を合計します。
func totalHistogramCount[N int64 | float64](points []metricdata.HistogramDataPoint[N]) uint64 {
	var total uint64
	for _, dp := range points {
		total += dp.Count
	}

	return total
}

// NewObservedHTTPClientMetrics は、計上内容を読み出せる HTTPClientMetrics を返します。
// 計上先の label を検証したい呼び出し元テストで使用します。
func NewObservedHTTPClientMetrics(t *testing.T) *ObservedHTTPClientMetrics {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	hm, err := NewHTTPClientMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatalf("failed to build observed http client metrics: %v", err)
	}

	return &ObservedHTTPClientMetrics{HTTPClientMetrics: hm, reader: reader}
}

// LabelValues は、metricName の counter に付いた labelKey の値を返します（未計上なら空）。
// 返るのは counter のデータポイント単位、すなわち計上された label 組であり、記録回数ではありません
// （同一 label 組へ何回記録しても cumulative counter の 1 データポイントに集約されます）。
// metricName が int64 counter でない場合は、テストキットの誤用としてテストを失敗させます。
func (o *ObservedHTTPClientMetrics) LabelValues(t *testing.T, metricName, labelKey string) []string {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := o.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect http client metrics: %v", err)
	}

	values, ok := counterLabelValues(rm, metricName, labelKey)
	if !ok {
		t.Fatalf("metric %s is not an int64 counter", metricName)
	}
	return values
}

// counterLabelValues は、rm から metricName の int64 counter を探し、各データポイントの labelKey の値を返します。
// metricName の指標が int64 counter でない場合は ok=false を返します（未計上の場合は空・ok=true）。
func counterLabelValues(rm metricdata.ResourceMetrics, metricName, labelKey string) ([]string, bool) {
	var values []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricName {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				return nil, false
			}
			for _, dp := range sum.DataPoints {
				if v, found := dp.Attributes.Value(attribute.Key(labelKey)); found {
					values = append(values, v.String())
				}
			}
		}
	}
	return values, true
}

// NewNoopOutboxMetrics は、テスト用に no-op の MeterProvider から OutboxMetrics を生成します。
func NewNoopOutboxMetrics(t *testing.T) *OutboxMetrics {
	t.Helper()
	om, err := NewOutboxMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop outbox metrics: %v", err)
	}
	return om
}

// NewNoopRealtimeMetrics は、テスト用に no-op の MeterProvider から RealtimeMetrics を生成します。
func NewNoopRealtimeMetrics(t *testing.T) *RealtimeMetrics {
	t.Helper()
	rm, err := NewRealtimeMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop realtime metrics: %v", err)
	}
	return rm
}

// NewNoopHTTPClientTransport は、テスト用に no-op TracerProvider と実 propagator から HTTPClientTransport を
// 生成します。SSRF ガードは無効化（loopback/httptest 宛てを許可）します。
func NewNoopHTTPClientTransport(t *testing.T) *HTTPClientTransport {
	t.Helper()
	return newHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator(), permissiveDialControl)
}

// NewGuardedHTTPClientTransport は、テスト用に no-op TracerProvider と実 propagator から
// HTTPClientTransport を生成します。NewNoopHTTPClientTransport と違い SSRF ガードは有効なままなので、
// private 網宛ての可否そのものを検証する用途に使います。
func NewGuardedHTTPClientTransport(t *testing.T) *HTTPClientTransport {
	t.Helper()
	return NewHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator())
}

// NewStubSpanContext は、テスト用のスタブSpanコンテキストを返します。
func NewStubSpanContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	tp := sdktrace.NewTracerProvider()
	tr := tp.Tracer(tracer)

	ctx, span := tr.Start(context.Background(), span)

	return ctx, func() {
		span.End()
		_ = tp.Shutdown(context.Background())
	}
}

func (r *spanRecorder) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spans = append(r.spans, spans...)
	return nil
}

func (r *spanRecorder) Shutdown(context.Context) error { return nil }

// NewRecordingTracerProvider は、終了した span を保持する TracerProvider と、保持した span を返す関数を返します。
// 計装が span の属性に何を載せたかをテストで検証するためのもので、テスト終了時に provider を停止します。
func NewRecordingTracerProvider(t *testing.T) (trace.TracerProvider, func() []sdktrace.ReadOnlySpan) {
	t.Helper()

	rec := &spanRecorder{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(rec))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

	return tp, func() []sdktrace.ReadOnlySpan {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return append([]sdktrace.ReadOnlySpan(nil), rec.spans...)
	}
}

// NewRecordingTracerFactory は、記録した span を読み戻せる TracerFactory と、その span を返す関数を返します。
func NewRecordingTracerFactory(t *testing.T) (TracerFactory, func() []sdktrace.ReadOnlySpan) {
	t.Helper()

	tp, recorded := NewRecordingTracerProvider(t)

	return NewTracerFactory(tp), recorded
}

// InstallRecordingTracerProvider は、NewRecordingTracerProvider の provider をプロセス全体の既定（otel の global）に
// 据え、テスト終了時に元へ戻します。global の provider から tracer を得る計装（HTTP の OTel ミドルウェアなど）が
// span に何を載せたかを、その計装自身を通して検証するために使います。
func InstallRecordingTracerProvider(t *testing.T) func() []sdktrace.ReadOnlySpan {
	t.Helper()

	tp, recorded := NewRecordingTracerProvider(t)
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	return recorded
}

// SpanAttributeValues は、属性 filterKey が filterValue に一致する span について、その全属性の値を文字列で返します。
// 計装が span に載せた値の中に特定の文字列（資格情報など）が無いことを表明するために使います。
func SpanAttributeValues(spans []sdktrace.ReadOnlySpan, filterKey, filterValue string) []string {
	var values []string
	for _, span := range spans {
		if !hasAttribute(span, filterKey, filterValue) {
			continue
		}
		for _, attr := range span.Attributes() {
			values = append(values, attr.Value.AsString())
		}
	}
	return values
}

// hasAttribute は、span が key に value を持つかを返します。
func hasAttribute(span sdktrace.ReadOnlySpan, key, value string) bool {
	for _, attr := range span.Attributes() {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return true
		}
	}
	return false
}
