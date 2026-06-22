package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

// run は、Engine.Run 1 回ぶんの実行状態です。
type run struct {
	e        *Engine
	name     string
	consumer worker.Consumer
	handler  worker.Handler
	failure  worker.FailureHandler

	cb        *circuit
	inflight  chan struct{} // in-flight 上限セマフォ（B2）
	conc      chan struct{} // 同時 Handle 上限セマフォ（B1）
	slotFreed chan struct{} // in-flight 解放の通知（poll loop 起床用）
	keyed     *keyedDispatcher
	wg        sync.WaitGroup

	cancel  context.CancelFunc
	fatalMu sync.Mutex
	fatal   error
}

func newRun(e *Engine, w worker.Worker) *run {
	set := e.set
	r := &run{
		e:         e,
		name:      w.Name(),
		consumer:  w.Consumer(),
		handler:   w.Handler(),
		failure:   w.FailureHandler(),
		cb:        newCircuit(set.CircuitFailureThreshold, set.circuitBackoff()),
		inflight:  make(chan struct{}, set.MaxInFlight),
		conc:      make(chan struct{}, set.Concurrency),
		slotFreed: make(chan struct{}, 1),
	}
	r.keyed = newKeyedDispatcher(set.MaxInFlight, r.process)
	return r
}

// loop は、poll loop 本体です。ctx 完了 or Fatal で抜け、drain して返ります。
func (r *run) loop(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	defer cancel()
	defer r.drain()

	for {
		n, ok := r.acquire(ctx)
		if !ok {
			return r.fatalErr()
		}
		msgs, err := r.consumer.Receive(ctx, n)
		if err != nil {
			if ctx.Err() != nil {
				return r.fatalErr()
			}
			r.onPollError(err)
			continue
		}
		r.dispatchAll(ctx, msgs)
	}
}

// acquire は、Receive を許可できる状態になるまで待ち、取得件数を返します。
// サーキットが Open なら cooldown を待ち、in-flight に空きが出るまで待つ二段ゲートです。
// スロット待ちの間に Open へ遷移しうるため、待機後に状態を再確認します。
func (r *run) acquire(ctx context.Context) (int, bool) {
	for {
		if ctx.Err() != nil {
			return 0, false
		}
		if r.cb.phaseNow() == phaseOpen {
			if !r.waitCooldown(ctx) {
				return 0, false
			}
			continue
		}
		free, ok := r.waitForSlot(ctx)
		if !ok {
			return 0, false
		}
		if r.cb.phaseNow() == phaseOpen {
			continue // スロット待ちの間に Open へ遷移したので Receive せず再評価
		}
		limit := r.e.set.BatchSize
		if r.cb.phaseNow() == phaseHalfOpen {
			limit = r.e.set.CircuitHalfOpenProbe
		}
		return min(limit, free), true
	}
}

// waitForSlot は、in-flight に空きが出るまで待ち、空き数を返します（B2 のゲート）。
func (r *run) waitForSlot(ctx context.Context) (int, bool) {
	for {
		free := cap(r.inflight) - len(r.inflight)
		if free > 0 {
			return free, true
		}
		select {
		case <-ctx.Done():
			return 0, false
		case <-r.slotFreed:
		}
	}
}

// waitCooldown は、Open の cooldown を待って Half-open へ遷移します。
func (r *run) waitCooldown(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(r.cb.cooldown()):
		r.cb.toHalfOpen()
		return true
	}
}

// dispatchAll は、受信メッセージを in-flight 計上のうえ dispatch へ回します。
func (r *run) dispatchAll(ctx context.Context, msgs []worker.Message) {
	for _, m := range msgs {
		r.inflight <- struct{}{}
		r.wg.Add(1)
		r.keyed.dispatch(ctx, m)
	}
}

// process は、1 メッセージの処理単位です（B1 の同時数制御・span・分類処理）。
func (r *run) process(ctx context.Context, m worker.Message) {
	defer r.finishMessage()

	ctx, endSpan := r.e.tracer.Start(ctx)
	defer endSpan()

	r.conc <- struct{}{}
	defer func() { <-r.conc }()

	r.warnIfPoison(m)
	err := r.safeHandle(ctx, m)
	r.handleResult(ctx, m, err)
}

// finishMessage は、in-flight トークンを解放し poll loop を起床させます。
func (r *run) finishMessage() {
	<-r.inflight
	select {
	case r.slotFreed <- struct{}{}:
	default:
	}
	r.wg.Done()
}

// safeHandle は、per-message recover（A6）と Extend ハートビート（A3）付きで Handle を実行します。
func (r *run) safeHandle(ctx context.Context, m worker.Message) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = xerrors.Wrap(apperror.ErrRetryable, fmt.Sprintf("panic recovered: %v", rec))
		}
	}()

	stop := r.startHeartbeat(ctx, m)
	defer stop()

	return r.handler.Handle(ctx, m)
}

// startHeartbeat は、ExtendInterval ごとに Extend を呼ぶハートビートを開始し、停止関数を返します。
func (r *run) startHeartbeat(ctx context.Context, m worker.Message) func() {
	interval := r.e.set.ExtendInterval
	if interval <= 0 {
		return func() {}
	}

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.consumer.Extend(ctx, m, interval*2)
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
	}
}

// handleResult は、Handle の結果を分類して Ack / Nack / FailureHandler / engine 停止に振り分けます。
func (r *run) handleResult(ctx context.Context, m worker.Message, err error) {
	if err == nil {
		r.ack(ctx, m) // A1: 成功時のみ Ack
		r.cb.onSuccess()
		return
	}

	switch classify(err) {
	case catRetryable:
		r.nack(ctx, m) // A2
		r.cb.onFailure()
	case catPermanent:
		r.routePermanent(ctx, m, err) // A5
		r.cb.onSuccess()
	case catFatal:
		r.triggerFatal(err) // A5（Fatal）
	}
}

// routePermanent は、Permanent を FailureHandler へ退避してから Ack します（退避失敗時は Ack しない）。
func (r *run) routePermanent(ctx context.Context, m worker.Message, cause error) {
	if r.failure != nil {
		if err := r.failure.Fail(ctx, m, cause); err != nil {
			r.logErr("worker.failure", "failure handler error", err)
			return
		}
	}
	r.ack(ctx, m)
}

func (r *run) ack(ctx context.Context, m worker.Message) {
	if err := r.consumer.Ack(ctx, m); err != nil {
		r.logErr("worker.ack", "ack error", err)
	}
}

func (r *run) nack(ctx context.Context, m worker.Message) {
	if err := r.consumer.Nack(ctx, m); err != nil {
		r.logErr("worker.nack", "nack error", err)
	}
}

// onPollError は、Receive の失敗をサーキットへ反映します（broker 到達不能など）。
func (r *run) onPollError(err error) {
	r.logErr("worker.poll", "receive error", err)
	r.cb.onFailure()
}

// warnIfPoison は、再配送回数が閾値以上のとき warn します（A7。無限ループは IaC の DLQ で打ち切る）。
func (r *run) warnIfPoison(m worker.Message) {
	th := r.e.set.ReceiveCountWarnThreshold
	if th <= 0 || m.ReceiveCount < th {
		return
	}
	r.e.log.Named("worker.poison").Warn(
		"receive count threshold reached",
		logging.String(logging.WorkerNameKey, r.name),
		logging.String(logging.MessageIDKey, m.ID),
		logging.Int(logging.ReceiveCountKey, m.ReceiveCount),
	)
}

// triggerFatal は、Fatal を記録して engine を停止（ctx キャンセル）します。
func (r *run) triggerFatal(err error) {
	r.fatalMu.Lock()
	if r.fatal == nil {
		r.fatal = err
	}
	r.fatalMu.Unlock()

	r.e.log.Named("worker.fatal").Error("fatal error, stopping engine", logging.Error(logging.ErrorKey, err))
	r.cancel()
}

func (r *run) fatalErr() error {
	r.fatalMu.Lock()
	defer r.fatalMu.Unlock()
	return r.fatal
}

func (r *run) logErr(name, msg string, err error) {
	r.e.log.Named(name).Error(msg, logging.Error(logging.ErrorKey, err))
}

// drain は、in-flight の完了を DrainTimeout まで待ちます（C1。未完は Ack されず再配送へ）。
func (r *run) drain() {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(r.e.set.DrainTimeout):
	}
}
