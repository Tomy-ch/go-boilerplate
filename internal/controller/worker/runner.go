package worker

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

// Engine は、選択された worker を pull-ack で実行する driving adapter です。
// seam（worker.Consumer / Handler / FailureHandler）のみに依存し、broker 実装には依存しません。
type Engine struct {
	workers map[string]worker.Worker
	set     Settings
	tracer  observability.LayerTracer
	log     logging.Logger
	met     *metrics

	active   atomic.Bool  // Run 実行中か
	progress atomic.Int64 // 最後に poll loop が進んだ時刻(UnixNano)
}

// New は、Engine を生成します。worker 名の重複は ErrDuplicateWorker を返します。
func New(
	workers []worker.Worker,
	set Settings,
	tf observability.TracerFactory,
	log logging.Logger,
) (*Engine, error) {
	m := make(map[string]worker.Worker, len(workers))
	for _, w := range workers {
		name := w.Name()
		if _, exists := m[name]; exists {
			return nil, xerrors.Wrap(ErrDuplicateWorker, name)
		}
		m[name] = w
	}
	set.normalize()

	met, err := newMetrics(otel.Meter(meterName))
	if err != nil {
		return nil, err
	}

	return &Engine{
		workers: m,
		set:     set,
		tracer:  tf.Controller(),
		log:     log,
		met:     met,
	}, nil
}

// Names は、登録されている worker 名の一覧を返します。
func (e *Engine) Names() []string {
	out := make([]string, 0, len(e.workers))
	for k := range e.workers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Run は、指定された worker を実行します。ctx 完了 or Fatal で drain して返ります。
// 通常（Retryable/Permanent のみ）の場合は ctx 完了まで常駐します。
func (e *Engine) Run(ctx context.Context, name string) error {
	w, ok := e.workers[name]
	if !ok {
		return xerrors.Wrap(ErrUnknownWorker, name)
	}

	e.active.Store(true)
	e.markProgress()
	defer e.active.Store(false)

	return newRun(e, w).loop(ctx)
}

// Healthy は、readiness 判定を返します（C2）。Run 実行中かつ poll 進捗が閾値内のとき true。
func (e *Engine) Healthy() bool {
	if !e.active.Load() {
		return false
	}
	last := time.Unix(0, e.progress.Load())
	return time.Since(last) < e.set.ProgressStaleAfter
}

// markProgress は、poll loop が進んだ時刻を記録します（stuck 検出の基準）。
func (e *Engine) markProgress() {
	e.progress.Store(time.Now().UnixNano())
}
