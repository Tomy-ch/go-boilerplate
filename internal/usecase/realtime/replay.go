//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"

	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// ReplayPageLimit は、1 回の読み取りで返す event の上限です。接続の送出 buffer はこれと同じ大きさを取り、
// 1 ページが buffer を溢れさせないという関係を保ちます。
const ReplayPageLimit = 64

// Replayer は、cursor より後ろの event を 1 ページずつ読みます。接続を開いたときの replay と、
// wakeup / 周期 catch-up による読み直しは同じ読み取りで、違うのは呼ぶ契機だけです。
type Replayer interface {
	// ReadPage は、after より後ろの event を sequence 昇順に最大 ReplayPageLimit 件返します。
	// 続きがあれば hasMore が true になり、呼び出し側は最後の event の Sequence を次の after に渡します。
	// EventLog が読めなければ apperror.ErrUnavailable（retry 可能）を返します。
	ReadPage(ctx context.Context, streamID rt.StreamID, after rt.Sequence) (events []rt.DeliveryEvent, hasMore bool, err error)
}

type replayer struct {
	log    rt.EventLogStore
	tracer observability.LayerTracer
}

// NewReplayer は、Replayer を生成します。
func NewReplayer(log rt.EventLogStore, tf observability.TracerFactory) Replayer {
	return &replayer{log: log, tracer: tf.Usecase()}
}

// ReadPage は、EventLog を 1 ページ読みます。gap の有無は判定しません — 連続かどうかは受け取った側が
// 自分の位置と突き合わせて決めることで、ここは「その位置より後ろにある物」をそのまま返します。
func (r *replayer) ReadPage(
	ctx context.Context, streamID rt.StreamID, after rt.Sequence,
) ([]rt.DeliveryEvent, bool, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	res, err := r.log.ReadAfter(ctx, rt.ReadAfterQuery{StreamID: streamID, After: after, Limit: ReplayPageLimit})
	if err != nil {
		return nil, false, err
	}

	return res.Events, res.HasMore, nil
}
