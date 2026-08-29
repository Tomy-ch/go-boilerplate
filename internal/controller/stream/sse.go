package stream

import (
	"encoding/json"
	"net/http"
	"time"

	"go-boilerplate/internal/controller/stream/gen"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// SSE のフレーム区切りと、business event 以外が id を持たないことを表す接頭辞。
const (
	frameEnd        = "\n\n"
	idField         = "id: "
	dataField       = "data: "
	controlPreamble = "event: control\n"
	heartbeatFrame  = ": heartbeat" + frameEnd
)

// sseWriter は、確定済みのレスポンスへ SSE のフレームを 1 つずつ書きます。
// 書き込みのたびに write deadline を張り直すので、接続そのものは maximum lifetime まで生き続け、
// 止まった peer だけが deadline で切れます（http.Server の WriteTimeout はレスポンス単位なので、
// 張り直さなければ 1 分あまりで stream 全体が切れます）。
type sseWriter struct {
	res      http.ResponseWriter
	ctrl     *http.ResponseController
	deadline time.Duration
}

// newSSEWriter は、res へ書く sseWriter を生成します。deadline は 1 回の書き込みに与える猶予です。
func newSSEWriter(res http.ResponseWriter, deadline time.Duration) *sseWriter {
	return &sseWriter{res: res, ctrl: http.NewResponseController(res), deadline: deadline}
}

// commit は、SSE のレスポンスヘッダを送ります。これ以降 HTTP ステータスは変えられず、
// client へ伝えられるのは in-band の control event と切断だけになります。
func (w *sseWriter) commit() error {
	h := w.res.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.res.WriteHeader(http.StatusOK)

	return w.flush()
}

// writeEvent は、business event を 1 フレーム書きます。SSE の id は sequence で、client の
// Last-Event-ID はこの値だけで進みます。
func (w *sseWriter) writeEvent(e rt.DeliveryEvent) error {
	body, err := e.MarshalJSON()
	if err != nil {
		return err
	}

	return w.write(idField + e.Sequence.String() + "\n" + dataField + string(body) + frameEnd)
}

// writeControl は、control event を 1 フレーム書きます。**id を持ちません** — 持たせると
// client の Last-Event-ID が制御指示で上書きされ、再接続の位置が business event の列から外れます。
func (w *sseWriter) writeControl(ev gen.ControlEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return xerrors.Wrap(err, "marshal control event")
	}

	return w.write(controlPreamble + dataField + string(body) + frameEnd)
}

// writeHeartbeat は、id を持たない comment を 1 フレーム書きます。到達しない peer をここで検出します。
func (w *sseWriter) writeHeartbeat() error {
	return w.write(heartbeatFrame)
}

// write は、deadline を張り直してから frame を書き、flush します。
func (w *sseWriter) write(frame string) error {
	if err := w.ctrl.SetWriteDeadline(time.Now().Add(w.deadline)); err != nil {
		return xerrors.Wrap(err, "set write deadline")
	}

	if _, err := w.res.Write([]byte(frame)); err != nil {
		return xerrors.Wrap(err, "write sse frame")
	}

	return w.flush()
}

// flush は、書いた分を client へ押し出します。proxy と Go の buffer を跨いで即時性を保ちます。
func (w *sseWriter) flush() error {
	if err := w.ctrl.Flush(); err != nil {
		return xerrors.Wrap(err, "flush sse frame")
	}

	return nil
}
