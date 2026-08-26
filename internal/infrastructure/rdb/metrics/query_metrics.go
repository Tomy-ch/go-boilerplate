package metrics

import (
	"context"
	"fmt"
	"strings"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/pkg/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	queryNamespace = "rdb"
	querySubsystem = "query"
)

// errIncompatibleCollector は、同名で登録済みのコレクタが期待する型と非互換だった場合のエラーです。
var errIncompatibleCollector = xerrors.New("registerOrExisting: collector is already registered with an incompatible type")

// queryMetrics は、DB クエリの duration / error を Prometheus メトリクスとして記録する driver.QueryRecorder 実装です。
type queryMetrics struct {
	duration    *prometheus.HistogramVec
	errorsTotal *prometheus.CounterVec
}

// NewQueryRecorder は、DB クエリメトリクス(rdb_query_duration_seconds / rdb_query_errors_total)を
// 指定レジストリに登録し、recorder を返します。既に登録済みの場合は既存のコレクタを再利用します。
func NewQueryRecorder(reg prometheus.Registerer) driver.QueryRecorder {
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: queryNamespace,
		Subsystem: querySubsystem,
		Name:      "duration_seconds",
		Help:      "Duration of DB queries in seconds, partitioned by query name, operation and status.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"query_name", "operation", "status"})

	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: queryNamespace,
		Subsystem: querySubsystem,
		Name:      "errors_total",
		Help:      "Total number of failed DB queries, partitioned by query name, operation and error class.",
	}, []string{"query_name", "operation", "error_class"})

	return &queryMetrics{
		duration:    registerOrExisting(reg, duration),
		errorsTotal: registerOrExisting(reg, errorsTotal),
	}
}

// Observe は、1 クエリの duration を記録し、エラー時のみ error counter を増分します。
func (m *queryMetrics) Observe(_ context.Context, attrs driver.QueryAttrs) {
	m.duration.
		WithLabelValues(attrs.QueryName, attrs.Operation, attrs.Status).
		Observe(attrs.Duration.Seconds())

	// ErrorClass が非空のときのみエラー（pgx.ErrNoRows を除く失敗）として計上します。
	if attrs.ErrorClass != "" {
		m.errorsTotal.WithLabelValues(attrs.QueryName, attrs.Operation, attrs.ErrorClass).Inc()
	}
}

// registerOrExisting は、コレクタを登録します。既に同一コレクタが登録済みの場合は、
// 新規生成したものではなく登録済みのコレクタを返し、複数回初期化されても同じメトリクスへ記録できるようにします。
//
// 「同名で既存だが型が一致しない」および AlreadyRegistered 以外の登録失敗（descriptor 不整合等）は、
// メトリクス初期化パスの設定バグです。黙って別インスタンスへ記録し続けると計測が二重化し発見が遅れるため、
// 起動時に panic で表面化させます（panic 値に対象コレクタの FQName を含めます）。
func registerOrExisting[T prometheus.Collector](reg prometheus.Registerer, c T) T {
	err := reg.Register(c)
	if err == nil {
		return c
	}

	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if xerrors.As(err, &alreadyRegisteredErr) {
		if existing, ok := alreadyRegisteredErr.ExistingCollector.(T); ok {
			return existing
		}
		panic(xerrors.Wrap(errIncompatibleCollector, fmt.Sprintf("collector [%s], existing type %T",
			collectorFQNames(c), alreadyRegisteredErr.ExistingCollector)))
	}
	panic(xerrors.Wrap(err, fmt.Sprintf(
		"registerOrExisting: failed to register collector [%s]", collectorFQNames(c))))
}

// collectorFQNames は、panic メッセージ用にコレクタの descriptor（FQName 含む）を文字列化します。
func collectorFQNames(c prometheus.Collector) string {
	ch := make(chan *prometheus.Desc, 1)
	go func() {
		c.Describe(ch)
		close(ch)
	}()

	descs := make([]string, 0, 1)
	for d := range ch {
		descs = append(descs, d.String())
	}
	return strings.Join(descs, ", ")
}
