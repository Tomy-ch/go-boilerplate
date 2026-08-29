package stream

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"

	"go-boilerplate/internal/controller/stream/gen"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/retry"
	"go-boilerplate/pkg/xerrors"
)

// 接続が閉じた理由。構造化ログに載せる安定値で、lifecycle metrics（#1417）はこの分岐を数えます。
const (
	// closeReasonClientGone は、client 側が読むのをやめた（切断・EOF）ことを表します。
	closeReasonClientGone = "client_gone"
	// closeReasonSlowClient は、client が追いつけず送出 buffer が溢れたか、書き込みが期限切れしたことを表します。
	closeReasonSlowClient = "slow_client"
	// closeReasonLifetime は、connection maximum lifetime に到達したことを表します。
	closeReasonLifetime = "lifetime"
	// closeReasonRevoked は、subject の権利が取り下げられたことを表します。
	closeReasonRevoked = "revoked"
	// closeReasonDraining は、instance が停止に入ったことを表します。
	closeReasonDraining = "draining"
	// closeReasonResync は、cursor の続きが EventLog に無く、正本からの取り直しが要ることを表します。
	closeReasonResync = "resync"
	// closeReasonCanceled は、request context が終わった（client 切断か shutdown）ことを表します。
	closeReasonCanceled = "canceled"
)

// connection は、確定済みの 1 本の SSE 接続です。読む側（fetcher）と書く側（pump）の 2 つの goroutine が
// channel だけで繋がり、位置を表す値はそれぞれの持ち主が 1 つずつ持ちます。
type connection struct {
	// id は、registry の索引に使う instance 内で一意な番号です。
	id uint64
	// subject は、ticket が束縛された subject です。失効通知はこれで接続を引きます。
	subject string
	// stream は、この接続が配信する stream です。wakeup はこれで接続を引きます。
	stream rt.StreamID

	// events は、読んだ event を送出まで待たせる buffer です。溢れたら接続を閉じます。
	events chan rt.DeliveryEvent
	// control は、client へ渡す最後の指示を 1 つだけ積みます。到達は保証されません（設計 §4.3）。
	control chan gen.ControlEvent
	// signal は、wakeup が届いたことを fetcher へ伝えます。
	signal chan struct{}
	// pending は、wakeup が伝えてきた最大の位置です。fetcher が追いついていれば読み直しません。
	pending atomic.Int64

	// quit は、接続を閉じると決まったときに閉じられます。
	quit chan struct{}
	// done は、pump が戻って接続が実際に終わったときに閉じられます。drain はこれを待ちます。
	done chan struct{}

	quitOnce sync.Once
	// reason は、最初に close を呼んだ側が記録した理由です。quitOnce の完了後にだけ読みます。
	reason string
}

// fetcher は、1 接続ぶんの読み直しを回します。読んだ位置はこの goroutine だけが持つので、
// 位置の同期は要りません。
type fetcher struct {
	conn     *connection
	replayer ucrealtime.Replayer
	sem      *semaphore.Weighted
	sleeper  clock.Sleeper
	log      logging.Logger
	// fetched は、buffer へ入れ終えた位置です。次の読み取りはこの後ろから始めます。
	fetched rt.Sequence
	// holdsSlot は、初回 replay のために Stream が確保した枠をまだ持っているかどうかです。
	holdsSlot bool
}

// newConnection は、subject × stream の接続を生成します。
func newConnection(id uint64, subject string, stream rt.StreamID) *connection {
	return &connection{
		id:      id,
		subject: subject,
		stream:  stream,
		events:  make(chan rt.DeliveryEvent, connectionBuffer),
		control: make(chan gen.ControlEvent, 1),
		signal:  make(chan struct{}, 1),
		quit:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// close は、接続を閉じると決めます。最初の呼び出しの reason だけが記録され、以降は無視されます。
func (c *connection) close(reason string) {
	c.quitOnce.Do(func() {
		c.reason = reason
		close(c.quit)
	})
}

// closeWith は、client へ渡す指示を積んでから接続を閉じます。理由の確定を指示の積載より前に置くので、
// pump が指示を先に拾って戻り、Stream の後片付けが先に close を呼んでも、記録される理由はこちらのままです。
func (c *connection) closeWith(ev gen.ControlEvent, reason string) {
	c.quitOnce.Do(func() {
		c.reason = reason
		c.signalControl(ev)
		close(c.quit)
	})
}

// signalControl は、client へ渡す指示を 1 つ積みます。既に積まれていれば捨てます。
func (c *connection) signalControl(ev gen.ControlEvent) {
	select {
	case c.control <- ev:
	default:
	}
}

// takeControl は、積まれている指示があれば取り出します。
func (c *connection) takeControl() (gen.ControlEvent, bool) {
	select {
	case ev := <-c.control:
		return ev, true
	default:
		return gen.ControlEvent{}, false
	}
}

// wake は、stream の位置が upTo まで進んだことを伝えます。重複と後戻りは冪等に吸収し、
// batch をまたぐ通知は pending の最大値へ畳まれます（ADR-0073）。
func (c *connection) wake(upTo rt.Sequence) {
	for {
		cur := c.pending.Load()
		if int64(upTo) <= cur {
			break
		}

		if c.pending.CompareAndSwap(cur, int64(upTo)) {
			break
		}
	}

	select {
	case c.signal <- struct{}{}:
	default:
	}
}

// startTicker は、sleeper で interval() ごとに tick を送る goroutine を起こします。
// tick は cap 1 の channel へ非ブロッキングに送るので、受け手が遅れても goroutine は詰まりません。
// ctx が終わると goroutine も終わります。
func startTicker(ctx context.Context, sleeper clock.Sleeper, interval func() time.Duration) <-chan struct{} {
	ch := make(chan struct{}, 1)

	go func() {
		for {
			if err := sleeper.Sleep(ctx, interval()); err != nil {
				return
			}

			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()

	return ch
}

// jitteredCatchUp は、次の周期 catch-up までの待ち時間を返します。全接続が同じ時刻に
// EventLog を読みに行かないよう、固定周期に full jitter を上乗せします。
func jitteredCatchUp() time.Duration {
	return catchUpInterval + retry.Full(catchUpJitter)
}

// writeFailureReason は、書き込みの失敗を close の理由に写します。期限切れは追いつけない client、
// それ以外は既に居ない client として扱います。
func writeFailureReason(err error) string {
	var netErr net.Error
	if xerrors.As(err, &netErr) && netErr.Timeout() {
		return closeReasonSlowClient
	}

	return closeReasonClientGone
}

// run は、初回 replay を Stream が確保した枠のまま走らせ、以降は wakeup と周期 catch-up で読み直します。
func (f *fetcher) run(ctx context.Context) {
	defer f.releaseSlot()

	f.drainPages(ctx)
	f.releaseSlot()

	catchUp := startTicker(ctx, f.sleeper, jitteredCatchUp)
	for {
		if !f.awaitTrigger(ctx, catchUp) {
			return
		}

		if err := f.sem.Acquire(ctx, 1); err != nil {
			return
		}

		f.drainPages(ctx)
		f.sem.Release(1)
	}
}

// awaitTrigger は、次に読み直す契機を待ちます。接続が終わるなら false を返します。
// wakeup は既に追いついていれば読み直さず、周期 catch-up は通知の欠落を埋めるため常に読みます。
func (f *fetcher) awaitTrigger(ctx context.Context, catchUp <-chan struct{}) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-f.conn.quit:
			return false
		case <-catchUp:
			return true
		case <-f.conn.signal:
			if f.conn.pending.Load() > int64(f.fetched) {
				return true
			}
		}
	}
}

// drainPages は、追いつくまで EventLog を読み進めます。semaphore の枠を持った状態で呼びます。
func (f *fetcher) drainPages(ctx context.Context) {
	for {
		events, hasMore, err := f.replayer.ReadPage(ctx, f.conn.stream, f.fetched)
		if err != nil {
			// 依存の不調でこの接続は閉じません。一斉に閉じると回復が再接続の嵐になるため、
			// 次の catch-up に委ねます（docs/design/realtime-delivery.md §2.6）。
			f.log.Warn(ctx, "failed to read the event log for a stream connection",
				logging.String(logging.StreamIDKey, string(f.conn.stream)),
				logging.Error(logging.ErrorKey, err))

			return
		}

		if len(events) == 0 {
			return
		}

		if events[0].Sequence != f.fetched+1 {
			f.conn.closeWith(resyncControl(), closeReasonResync)

			return
		}

		if !f.push(events) || !hasMore {
			return
		}
	}
}

// push は、読んだ event を送出 buffer へ入れます。満杯なら RETRY_LATER を試みて接続を閉じます。
// event 自体は EventLog に残るので失われず、再接続時の replay が回収します。
func (f *fetcher) push(events []rt.DeliveryEvent) bool {
	for _, e := range events {
		select {
		case f.conn.events <- e:
			f.fetched = e.Sequence
		case <-f.conn.quit:
			return false
		default:
			f.conn.closeWith(retryLaterControl(), closeReasonSlowClient)

			return false
		}
	}

	return true
}

// releaseSlot は、初回 replay の枠を返します。2 度目以降は何もしません。
func (f *fetcher) releaseSlot() {
	if !f.holdsSlot {
		return
	}

	f.holdsSlot = false
	f.sem.Release(1)
}

// control 指示の構築子。action と reason の対応はここが唯一の定義で、client 契約
// （docs/design/realtime-delivery.md §4.3）の reason code に 1 対 1 で対応します。
// STREAM_RECOVERY_FAILED は意図的に現れません（README「Reason codes this package sends」参照）。

// reconnectControl は、instance が停止に入ったので繋ぎ直すよう伝えます。
func reconnectControl() gen.ControlEvent {
	return gen.ControlEvent{Action: gen.RECONNECT, Reason: gen.SERVERDRAINING}
}

// retryLaterControl は、送出が追いつかないので間を置いて繋ぎ直すよう伝えます。
func retryLaterControl() gen.ControlEvent {
	ms := int(retryAfterHint.Milliseconds())

	return gen.ControlEvent{Action: gen.RETRYLATER, Reason: gen.TEMPORARILYOVERLOADED, RetryAfterMs: &ms}
}

// reauthenticateControl は、接続の寿命が尽きたので新しい ticket で繋ぎ直すよう伝えます。
func reauthenticateControl() gen.ControlEvent {
	return gen.ControlEvent{Action: gen.REAUTHENTICATE, Reason: gen.AUTHREFRESHREQUIRED}
}

// resyncControl は、cursor の続きが EventLog に無いので正本を取り直すよう伝えます。
func resyncControl() gen.ControlEvent {
	return gen.ControlEvent{Action: gen.RESYNC, Reason: gen.CURSORTOOOLD}
}

// stopControl は、権利が取り下げられたので繋ぎ直さないよう伝えます。
func stopControl() gen.ControlEvent {
	return gen.ControlEvent{Action: gen.STOP, Reason: gen.AUTHORIZATIONREVOKED}
}
