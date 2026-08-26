//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package redmetrics は、HTTP の RED（Rate / Errors / Duration）メトリクスを
// Echo ミドルウェアとして計測するための実装を提供します。
// label は method / route / status_code / status_class のみで、raw path や query
// など高カーディナリティな値・秘匿情報は持たせません。
package redmetrics

import (
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "http"
	subsystem = "server"
)

// labelNames は、RED メトリクスに付与する label 名の一覧です。
var labelNames = []string{"method", "route", "status_code", "status_class"}

// Recorder は、1 リクエスト分の計測値を記録するためのインターフェースです。
type Recorder interface {
	// Observe は、1 リクエスト分の count と duration を記録します。
	Observe(method, route string, statusCode int, statusClass string, duration time.Duration)
}

// PrometheusRecorder は、Recorder を Prometheus メトリクスとして公開する実装です。
// prometheus.Collector も満たし、pool_stats_collector と同じく registry へ登録して使用します。
type PrometheusRecorder struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewPrometheusRecorder は、PrometheusRecorder を初期化して返します。
// 副作用は持たず、registry への登録は RegisterRecorder で別途行います。
func NewPrometheusRecorder() *PrometheusRecorder {
	return &PrometheusRecorder{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP server requests.",
		}, labelNames),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP server request latencies in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, labelNames),
	}
}

// Observe は、request count を increment し、duration を histogram に記録します。
func (r *PrometheusRecorder) Observe(method, route string, statusCode int, statusClass string, duration time.Duration) {
	labels := prometheus.Labels{
		"method":       method,
		"route":        route,
		"status_code":  statusCodeLabel(statusCode),
		"status_class": statusClass,
	}
	r.requests.With(labels).Inc()
	r.duration.With(labels).Observe(duration.Seconds())
}

// Describe は、PrometheusRecorder のメトリクスの説明を Prometheus に提供します。
func (r *PrometheusRecorder) Describe(ch chan<- *prometheus.Desc) {
	r.requests.Describe(ch)
	r.duration.Describe(ch)
}

// Collect は、PrometheusRecorder のメトリクスを収集して Prometheus に提供します。
func (r *PrometheusRecorder) Collect(ch chan<- prometheus.Metric) {
	r.requests.Collect(ch)
	r.duration.Collect(ch)
}

// RegisterRecorder は、PrometheusRecorder を指定レジストリに登録します。
// 既に登録済みの場合はエラーを返さず無視します。
func RegisterRecorder(reg prometheus.Registerer, r *PrometheusRecorder) error {
	return ignoreAlreadyRegistered(reg.Register(r))
}

// ignoreAlreadyRegistered は、AlreadyRegisteredError を無視し、それ以外のエラーはそのまま返します。
func ignoreAlreadyRegistered(err error) error {
	if err == nil {
		return nil
	}

	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if xerrors.As(err, &alreadyRegisteredErr) {
		return nil
	}
	return err
}
