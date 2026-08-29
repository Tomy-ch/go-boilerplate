package stream

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"

	"github.com/labstack/echo/v5"
)

// registryLoggerName は、接続の出来事を出すロガーの名前です。
const registryLoggerName = "realtime-stream"

var _ Streamer = (*Registry)(nil)

// Registry は、この instance が保持している SSE 接続の索引です。確定済みの接続を配信し続ける Streamer
// であると同時に、fan-out で届いた wakeup / 失効を接続へ渡す受け口（controller/realtime の Waker /
// Revoker）と、停止時に接続を閉じ切る drain を兼ねます。3 つの入口はどれも同じ索引を見るので、
// 型を分けると lock を跨いだ整合が要り、分けない理由がそのまま責務になります。
//
// 索引は 2 本あります。wakeup は stream 単位、失効は subject 単位で接続を引くためで、
// どちらも Realtime Delivery 中立の語彙です（誰が user で誰が operator かは知りません）。
type Registry struct {
	replayer ucrealtime.Replayer
	sleeper  clock.Sleeper
	logging  logging.Logger
	tracer   observability.LayerTracer
	set      Settings
	// sem は、初回 replay と catch-up が共有する同時実行数の枠です。
	sem *semaphore.Weighted

	mu sync.Mutex
	// draining は、停止に入って新規接続を受け付けない状態かどうかです。
	draining bool
	// nextID は、接続に振る instance 内で一意な番号の種です。
	nextID uint64
	conns  map[uint64]*connection
	// byStream は、wakeup が接続を引くための索引です。
	byStream map[rt.StreamID]map[uint64]*connection
	// bySubject は、失効通知が接続を引くための索引です。
	bySubject map[string]map[uint64]*connection
}

// NewRegistry は、接続の索引を生成します。set のゼロ値は既定値に寄せられます。
func NewRegistry(
	replayer ucrealtime.Replayer,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
	set Settings,
) *Registry {
	set = set.withDefaults()

	return &Registry{
		replayer:  replayer,
		sleeper:   sleeper,
		logging:   log.Named(registryLoggerName),
		tracer:    tf.Controller(),
		set:       set,
		sem:       semaphore.NewWeighted(int64(set.ReplayConcurrency)),
		conns:     map[uint64]*connection{},
		byStream:  map[rt.StreamID]map[uint64]*connection{},
		bySubject: map[string]map[uint64]*connection{},
	}
}

// Stream は、検証を通った接続にレスポンスを確定し、閉じるまで event を書き続けます。
// 拒否はすべて確定より前に返し、確定後は nil を返します — 確定後にエラーを返すと、
// 共有 error handler が書き込み済みのレスポンスへ本文を足そうとするためです。
func (r *Registry) Stream(c *echo.Context, req StreamRequest) error {
	ctx := c.Request().Context()

	conn, err := r.register(req)
	if err != nil {
		hintRetryAfter(c)

		return err
	}
	defer r.unregister(ctx, conn)

	// 索引へ載せた直後に ticket を検証し直します。ここより前に無効化されていれば拒否になり、
	// 後に無効化されるなら接続は既に索引に居るので失効通知が拾えます（ADR-0074）。
	if req.Revalidate != nil {
		if err := req.Revalidate(ctx); err != nil {
			return err
		}
	}

	// 初回 replay の枠は確定より前に取ります。確定後に待つと「繋がったのに何も来ない」接続になります。
	if err := r.admit(ctx); err != nil {
		hintRetryAfter(c)

		return err
	}

	w := newSSEWriter(c.Response(), writeDeadline)
	if err := w.commit(); err != nil {
		r.sem.Release(1)

		return err
	}

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go (&fetcher{
		conn: conn, replayer: r.replayer, sem: r.sem, sleeper: r.sleeper,
		log: r.logging, fetched: req.Cursor, holdsSlot: true,
	}).run(connCtx)

	r.pump(connCtx, conn, w)

	return nil
}

// Wake は、stream の位置が upTo まで進んだことを、その stream の接続へ伝えます。
// 重複した通知は正常で、既に追いついている接続は読み直しません。engine の loop を止めないよう、
// ここでは印を付けるだけで読み取りは待ちません。
func (r *Registry) Wake(_ context.Context, streamID rt.StreamID, upTo rt.Sequence) {
	r.mu.Lock()
	conns := values(r.byStream[streamID])
	r.mu.Unlock()

	for _, conn := range conns {
		conn.wake(upTo)
	}
}

// Revoke は、subject が destination への権利を失ったことを受けて、該当する接続を STOP で閉じます。
// ticket の無効化は既に済んでいるので、STOP を無視した client も再接続できません。
func (r *Registry) Revoke(_ context.Context, subject string, destination rt.StreamID) {
	r.mu.Lock()
	conns := values(r.bySubject[subject])
	r.mu.Unlock()

	for _, conn := range conns {
		if conn.stream != destination {
			continue
		}

		conn.closeWith(stopControl(), closeReasonRevoked)
	}
}

// Drain は、新規接続を止め、確定済みの接続に RECONNECT を送って閉じ切るまで待ちます。
// 待つのは停止 ctx の残りと drainBudget の短いほうで、超えた分は諦めます — ここで粘ると、
// 常駐処理の停止と instance resource の片付けに残す時間が無くなります。
func (r *Registry) Drain(ctx context.Context) error {
	conns := r.startDraining()

	for _, conn := range conns {
		conn.closeWith(reconnectControl(), closeReasonDraining)
	}

	budget, cancel := context.WithTimeout(ctx, drainBudget)
	defer cancel()

	for i, conn := range conns {
		select {
		case <-conn.done:
		case <-budget.Done():
			r.logging.Warn(ctx, "gave up draining stream connections within the budget",
				logging.Int64(logging.OpenConnectionsKey, int64(len(conns)-i)))

			return nil
		}
	}

	return nil
}

// pump は、確定済みの接続へ書き続けます。戻るときが接続の終わりです。
// 指示は event より先に届けます — STOP が buffer に溜まった 64 件の後ろで待つ状態を作らないためです。
func (r *Registry) pump(ctx context.Context, conn *connection, w *sseWriter) {
	heartbeat := startTicker(ctx, r.sleeper, func() time.Duration { return heartbeatInterval })
	lifetime := startTicker(ctx, r.sleeper, func() time.Duration { return maxConnectionLifetime })

	for {
		if ev, ok := conn.takeControl(); ok {
			_ = w.writeControl(ev)

			return
		}

		select {
		case ev := <-conn.control:
			_ = w.writeControl(ev)

			return
		case e := <-conn.events:
			if !r.writeOrClose(conn, w.writeEvent(e)) {
				return
			}
		case <-heartbeat:
			if !r.writeOrClose(conn, w.writeHeartbeat()) {
				return
			}
		case <-lifetime:
			_ = w.writeControl(reauthenticateControl())
			conn.close(closeReasonLifetime)

			return
		case <-conn.quit:
			if ev, ok := conn.takeControl(); ok {
				_ = w.writeControl(ev)
			}

			return
		case <-ctx.Done():
			conn.close(closeReasonCanceled)

			return
		}
	}
}

// writeOrClose は、書き込みの結果を接続の継続可否に変えます。失敗は client 側の事情なので、
// 期限切れか切断かだけを記録して閉じます。
func (r *Registry) writeOrClose(conn *connection, err error) bool {
	if err == nil {
		return true
	}

	conn.close(writeFailureReason(err))

	return false
}

// register は、接続を索引に加えます。停止中と上限到達は、レスポンス確定前に返せる唯一の機会です。
func (r *Registry) register(req StreamRequest) (*connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draining {
		return nil, ErrDraining
	}

	if len(r.conns) >= r.set.MaxConnections {
		return nil, ErrConnectionCapacity
	}

	r.nextID++
	conn := newConnection(r.nextID, req.Subject, req.Destination)
	r.conns[conn.id] = conn
	addIndex(r.byStream, conn.stream, conn)
	addIndex(r.bySubject, conn.subject, conn)

	return conn, nil
}

// unregister は、接続を索引から外して容量を返し、drain の待ち手に終わりを知らせます。
// close / cancel / 異常切断のどれで終わっても必ずここを通ります。
func (r *Registry) unregister(ctx context.Context, conn *connection) {
	conn.close(closeReasonCanceled)

	r.mu.Lock()
	delete(r.conns, conn.id)
	removeIndex(r.byStream, conn.stream, conn.id)
	removeIndex(r.bySubject, conn.subject, conn.id)
	r.mu.Unlock()

	r.logging.Info(ctx, "stream connection closed",
		logging.String(logging.StreamIDKey, string(conn.stream)),
		logging.String(logging.CloseReasonKey, conn.reason))

	close(conn.done)
}

// admit は、初回 replay の枠を有界時間だけ待って確保します。取れなければ 503 で断ります。
func (r *Registry) admit(ctx context.Context) error {
	waitCtx, cancel := context.WithTimeout(ctx, admissionWait)
	defer cancel()

	if err := r.sem.Acquire(waitCtx, 1); err != nil {
		return ErrReplayAdmission
	}

	return nil
}

// startDraining は、新規接続の受付を止め、そのとき保持していた接続を返します。
func (r *Registry) startDraining() []*connection {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.draining = true
	conns := make([]*connection, 0, len(r.conns))
	for _, conn := range r.conns {
		conns = append(conns, conn)
	}

	return conns
}

// values は、索引の 1 区画を slice にします。
func values(m map[uint64]*connection) []*connection {
	conns := make([]*connection, 0, len(m))
	for _, conn := range m {
		conns = append(conns, conn)
	}

	return conns
}

// addIndex は、索引に接続を加えます。
func addIndex[K comparable](m map[K]map[uint64]*connection, key K, conn *connection) {
	if m[key] == nil {
		m[key] = map[uint64]*connection{}
	}

	m[key][conn.id] = conn
}

// removeIndex は、索引から接続を外します。区画が空になったら区画ごと消します
// （stream も subject も数に限りが無いので、残すと索引が単調増加します）。
func removeIndex[K comparable](m map[K]map[uint64]*connection, key K, id uint64) {
	part, ok := m[key]
	if !ok {
		return
	}

	delete(part, id)
	if len(part) == 0 {
		delete(m, key)
	}
}
