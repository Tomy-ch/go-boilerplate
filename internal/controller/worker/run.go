package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/retry"
	"go-boilerplate/pkg/xerrors"
)

// extendVisibilityFactor は、Extend で延長する可視性を ExtendInterval の何倍にするかの係数です。
// ハートビート周期より長く延長して、tick 間に lease 切れ（早期再配送）が起きないようにします。
const extendVisibilityFactor = 2

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
		r.e.markProgress() // stuck 検出用に進捗時刻を更新
		n, ok := r.acquire(ctx)
		if !ok {
			return r.fatalErr()
		}
		msgs, err := r.consumer.Receive(ctx, n)
		if err != nil {
			if ctx.Err() != nil {
				return r.fatalErr()
			}
			r.onPollError(ctx, err)
			continue
		}
		if len(msgs) == 0 {
			// Half-open の probe が空振りしたので probing を解除し、次周で再 probe させる。
			r.cb.abortProbe()
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
		// phase は 1 度だけ snapshot して判定する（複数回 phaseNow() すると
		// その間の trip() 割り込みで Open なのに BatchSize 件 Receive しうる TOCTOU を防ぐ）。
		switch r.cb.phaseNow() {
		case phaseOpen:
			continue // スロット待ちの間に Open へ遷移したので Receive せず再評価
		case phaseHalfOpen:
			if r.cb.tryBeginProbe() {
				return min(r.e.set.CircuitHalfOpenProbe, free), true
			}
			// 既に probe 投入済み（probing 中）。結果が確定して Closed/Open へ遷移するまで新規 Receive を
			// 止め、in-flight 解放（probe 結果の到来で slotFreed が鳴る）で起床して再評価する。
			select {
			case <-ctx.Done():
				return 0, false
			case <-r.slotFreed:
				r.e.markProgress() // probing 待機からの起床は進捗とみなし、probe 待ちの間の readiness stale を避ける
			}
			continue
		default:
			return min(r.e.set.BatchSize, free), true
		}
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
	if len(msgs) == 0 {
		return
	}
	r.e.met.Received(ctx, int64(len(msgs)))
	for _, m := range msgs {
		r.inflight <- struct{}{}
		r.e.met.InFlightAdd(ctx, 1)
		r.wg.Add(1)
		r.keyed.dispatch(ctx, m)
	}
}

// process は、1 メッセージの処理単位です（B1 の同時数制御・span・分類処理）。
func (r *run) process(ctx context.Context, m worker.Message) {
	defer r.finishMessage(ctx)

	ctx = r.withTrace(ctx, m) // D1: traceparent から trace 継続
	ctx, endSpan := r.e.tracer.Start(ctx)
	defer endSpan()

	select {
	case r.conc <- struct{}{}:
	case <-ctx.Done():
		return // 停止中は未処理のまま離脱（Ack/Nack せず再配送へ）。in-flight は finishMessage が解放
	}
	defer func() { <-r.conc }()

	r.warnIfPoison(ctx, m)
	start := time.Now()
	err := r.safeHandle(ctx, m)
	r.e.met.RecordLatencyMs(ctx, msSince(start))
	r.handleResult(ctx, m, err)
}

// finishMessage は、in-flight トークンを解放し poll loop を起床させます。
func (r *run) finishMessage(ctx context.Context) {
	<-r.inflight
	r.e.met.InFlightAdd(ctx, -1)
	select {
	case r.slotFreed <- struct{}{}:
	default:
	}
	r.wg.Done()
}

// msSince は、start からの経過時間をミリ秒（小数）で返します。
func msSince(start time.Time) float64 {
	return float64(time.Since(start)) / float64(time.Millisecond)
}

// safeHandle は、per-message recover（A6）と Extend ハートビート（A3）付きで Handle を実行します。
func (r *run) safeHandle(ctx context.Context, m worker.Message) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// panic 値はログにのみ残し、伝播するエラーには含めない（秘密情報の漏洩防止）。
			r.e.log.Named("worker.recover").Error(
				ctx,
				"panic recovered in handler",
				append(msgFields(r.name, m), logging.String(logging.PanicKey, fmt.Sprintf("%v", rec)))...,
			)
			err = xerrors.Wrap(apperror.ErrRetryable, "panic recovered in handler")
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
	// stopped は goroutine の終了を通知する。停止クロージャがこれを待つことで、worker 停止後に
	// heartbeat goroutine が残らないこと（graceful-stop 契約）を保証する。
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.consumer.Extend(ctx, m, interval*extendVisibilityFactor); err != nil {
					if ctx.Err() != nil {
						return // 停止中の Extend 失敗は握り潰してよい（未 Ack なので再配送される）
					}
					// 握り潰さず可視化する。lease 延長失敗は早期再配送→重複処理の予兆。
					r.e.met.ExtendError(ctx)
					r.e.log.Named("worker.extend").Warn(
						ctx,
						"extend failed",
						append(msgFields(r.name, m), logging.Error(logging.ErrorKey, err))...,
					)
				}
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
		<-stopped // goroutine の完全終了を待つ（停止後に heartbeat が残らないことを保証）
	}
}

// handleResult は、Handle の結果を分類して Ack / Nack / FailureHandler / engine 停止に振り分けます。
// 併せて engine 所有 metric の更新（D2）と構造化ログ（D3）を行います。
func (r *run) handleResult(ctx context.Context, m worker.Message, err error) {
	if err == nil {
		r.ack(ctx, m) // A1: 成功時のみ Ack
		r.cb.onSuccess()
		r.e.met.Processed(ctx)
		r.e.log.Named("worker.process").Debug(ctx, "message processed", msgFields(r.name, m)...)
		return
	}

	r.e.met.Failed(ctx)
	fields := append(msgFields(r.name, m), logging.Error(logging.ErrorKey, err))

	switch classify(err) {
	case catRetryable:
		r.nack(ctx, m) // A2
		r.cb.onFailure()
		r.e.met.Retried(ctx)
		r.e.log.Named("worker.process").Warn(ctx, "retryable failure, nacked", fields...)
	case catPermanent:
		if ferr := r.routePermanent(ctx, m, err); ferr != nil { // A5
			// dead-letter 退避に失敗した。実際には退避できていないため DLQ 計上・成功ログは出さず、
			// Ack もしない（暗黙の再配送へ委ねる）。退避先障害は下流失敗として circuit へ計上する。
			r.cb.onFailure()
			r.e.log.Named("worker.process").Error(ctx, "permanent failure, dead-letter routing failed", fields...)
			return
		}
		r.cb.onSuccess()
		r.e.met.DLQ(ctx)
		r.e.log.Named("worker.process").Warn(ctx, "permanent failure, routed to dead-letter", fields...)
	case catFatal:
		r.e.log.Named("worker.process").Error(ctx, "fatal failure, stopping engine", fields...)
		r.triggerFatal(err) // A5（Fatal）
	}
}

// routePermanent は、Permanent を FailureHandler へ退避してから Ack します。
// 退避に失敗した場合は Ack せず、その error を返します（呼び出し元が circuit / ログを分岐する）。
func (r *run) routePermanent(ctx context.Context, m worker.Message, cause error) error {
	if r.failure != nil {
		if err := r.failure.Fail(ctx, m, cause); err != nil {
			r.logErr(ctx, "worker.failure", "failure handler error", err)
			return err
		}
	}
	r.ack(ctx, m)
	return nil
}

func (r *run) ack(ctx context.Context, m worker.Message) {
	if err := r.consumer.Ack(ctx, m); err != nil {
		r.logErr(ctx, "worker.ack", "ack error", err)
	}
}

// nack は、retryable 失敗時に per-message 再配送 backoff（指数 + full jitter）を計算し、
// その遅延つきで再配送します。
func (r *run) nack(ctx context.Context, m worker.Message) {
	d := r.nackBackoff(m.ReceiveCount)
	if err := r.consumer.NackWithBackoff(ctx, m, d); err != nil {
		r.logErr(ctx, "worker.nack", "nack error", err)
	}
}

// nackBackoff は、ReceiveCount（1 起算）に対する再配送遅延を返します。
// 指数 backoff を full jitter で散らし、同時失敗の thundering herd を避けます。
func (r *run) nackBackoff(receiveCount int) time.Duration {
	// ReceiveCount は 1 起算、backoff.Duration は 0 起算。
	attempt := max(0, receiveCount-1)
	return retry.Full(r.e.set.nackBackoff().Duration(attempt))
}

// onPollError は、Receive の失敗をサーキットへ反映します（broker 到達不能など）。
func (r *run) onPollError(ctx context.Context, err error) {
	r.e.met.PollError(ctx)
	r.logErr(ctx, "worker.poll", "receive error", err)
	r.cb.onFailure()
}

// warnIfPoison は、再配送回数が閾値以上のとき warn します（A7。無限ループは IaC の DLQ で打ち切る）。
func (r *run) warnIfPoison(ctx context.Context, m worker.Message) {
	th := r.e.set.ReceiveCountWarnThreshold
	if th <= 0 || m.ReceiveCount < th {
		return
	}
	r.e.log.Named("worker.poison").Warn(ctx, "receive count threshold reached", msgFields(r.name, m)...)
}

// triggerFatal は、Fatal を記録して engine を停止（ctx キャンセル）します。ログは handleResult が出します。
func (r *run) triggerFatal(err error) {
	r.fatalMu.Lock()
	if r.fatal == nil {
		r.fatal = err
	}
	r.fatalMu.Unlock()
	r.cancel()
}

func (r *run) fatalErr() error {
	r.fatalMu.Lock()
	defer r.fatalMu.Unlock()
	return r.fatal
}

func (r *run) logErr(ctx context.Context, name, msg string, err error) {
	r.e.log.Named(name).Error(ctx, msg, logging.Error(logging.ErrorKey, err))
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
