package realtime

import (
	"context"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

const (
	engineLoggerName = "realtime-consumer"

	// DefaultBatchSize は、1 回の受信で取り出す通知の上限です（受信先の上限 10 に揃える）。
	DefaultBatchSize = 10
	// DefaultErrorBackoff は、受信に失敗した後に次の受信まで待つ時間です。
	DefaultErrorBackoff = 5 * time.Second
)

// Settings は、consumer engine のチューニング値です。ゼロ値は既定値に寄せます。
type Settings struct {
	// BatchSize は、1 回の受信で取り出す通知の上限です。
	BatchSize int
	// ErrorBackoff は、受信に失敗した後に待つ時間です。
	ErrorBackoff time.Duration
}

// Engine は、instance の受信先から通知を取り出し、種別ごとに sink へ渡して削除する常駐 engine です。
type Engine struct {
	sub         rt.InstanceSubscription
	reprovision Reprovisioner
	fanout      FanoutObserver
	wakeups     Waker
	revocations Revoker
	sleeper     clock.Sleeper
	logging     logging.Logger
	tracer      observability.LayerTracer
	metrics     *observability.RealtimeMetrics
	set         Settings
}

// NewEngine は、consumer engine を生成します。
func NewEngine(
	sub rt.InstanceSubscription,
	reprovision Reprovisioner,
	fanout FanoutObserver,
	wakeups Waker,
	revocations Revoker,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
	metrics *observability.RealtimeMetrics,
	set Settings,
) *Engine {
	if set.BatchSize <= 0 {
		set.BatchSize = DefaultBatchSize
	}

	if set.ErrorBackoff <= 0 {
		set.ErrorBackoff = DefaultErrorBackoff
	}

	return &Engine{
		sub: sub, reprovision: reprovision, fanout: fanout, wakeups: wakeups, revocations: revocations,
		sleeper: sleeper, logging: log, tracer: tf.Controller(), metrics: metrics, set: set,
	}
}

// Run は、受信ループ本体です。ctx 完了まで常駐し、完了時に nil を返します。
// loop の意味論（受信失敗時の待機・削除順序・重複の扱い）は README の "Loop semantics (Run)" を参照。
func (e *Engine) Run(ctx context.Context) error {
	ctx, endSpan := e.tracer.Start(ctx)
	defer endSpan()

	log := e.logging.Named(engineLoggerName)
	for {
		if ctx.Err() != nil {
			return nil
		}

		batch, err := e.sub.Receive(ctx, e.set.BatchSize)
		e.fanout.ObserveFanout(err)

		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			log.Error(ctx, "failed to receive realtime notifications", logging.Error(logging.ErrorKey, err))
			e.repairIfGone(ctx, log, err)
			if e.sleeper.Sleep(ctx, e.set.ErrorBackoff) != nil {
				return nil
			}

			continue
		}

		e.dispatch(ctx, log, batch)
	}
}

// repairIfGone は、受信先が使えないことを示す失敗なら作り直しを 1 度試みます。失敗しても次の周回で同じ
// 失敗を踏んで再び試みるので、ここで諦めても復旧の機会は失われません（AWS は queue の削除から同名 queue の
// 作成まで 60 秒を要求するため、最初の数周は失敗するのが正常）。
func (e *Engine) repairIfGone(ctx context.Context, log logging.Logger, cause error) {
	if !xerrors.Is(cause, rt.ErrReceivingEndGone) {
		return
	}

	if err := e.reprovision.Reprovision(ctx); err != nil {
		if ctx.Err() != nil {
			return
		}

		log.Error(ctx, "failed to reprovision the realtime receiving end", logging.Error(logging.ErrorKey, err))
		e.metrics.RecoveryExecuted(ctx, observability.RealtimeResultError)

		return
	}

	log.Info(ctx, "reprovisioned the realtime receiving end")
	e.metrics.RecoveryExecuted(ctx, observability.RealtimeResultOK)
}

// dispatch は、1 回の受信分を種別ごとに sink へ渡し、渡し終えた通知を削除します。
func (e *Engine) dispatch(ctx context.Context, log logging.Logger, batch []rt.Notification) {
	if len(batch) == 0 {
		return
	}

	wakeups, revocations, unknown := coalesce(batch)
	if unknown > 0 {
		// 種別が読めない通知は誰も処理できず、残すと再配送され続けるので、記録して削除する。
		log.Warn(ctx, "discarding realtime notifications of unknown kind", logging.Int("count", unknown))
	}

	for streamID, upTo := range wakeups {
		e.wakeups.Wake(ctx, streamID, upTo)
	}

	for _, r := range revocations {
		e.revocations.Revoke(ctx, r.Subject, r.Destination)
	}

	for _, n := range batch {
		if err := e.sub.Delete(ctx, n); err != nil && ctx.Err() == nil {
			log.Error(ctx, "failed to delete realtime notification", logging.Error(logging.ErrorKey, err))
		}
	}
}

// coalesce は、1 回の受信分を種別ごとにまとめます。同じ stream の wakeup は最大の sequence 1 件に畳み、
// 失効通知はそのまま並べ、種別の無い通知は数だけ返します。
func coalesce(batch []rt.Notification) (map[rt.StreamID]rt.Sequence, []rt.Revocation, int) {
	wakeups := map[rt.StreamID]rt.Sequence{}
	revocations := make([]rt.Revocation, 0, len(batch))
	unknown := 0

	for _, n := range batch {
		switch n.Kind {
		case rt.KindWakeup:
			if cur, ok := wakeups[n.Wakeup.StreamID]; !ok || n.Wakeup.Sequence > cur {
				wakeups[n.Wakeup.StreamID] = n.Wakeup.Sequence
			}
		case rt.KindRevocation:
			revocations = append(revocations, n.Revocation)
		default:
			unknown++
		}
	}

	return wakeups, revocations, unknown
}
