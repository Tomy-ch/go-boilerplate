package worker

import (
	"context"
	"sort"

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

	return &Engine{
		workers: m,
		set:     set,
		tracer:  tf.Controller(),
		log:     log,
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
	return newRun(e, w).loop(ctx)
}
