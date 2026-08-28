//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package realtime は、Realtime Delivery の機構側 usecase — cursor の失効判定と ticket の発行 / 検証 —
// を提供します。feature の語彙は持たず、boundary/realtime にだけ依存します。replay の読み取りと
// lease の heartbeat / orphan cleanup の呼び出し側は、それぞれ stream handler と serve lifecycle が持ちます。
package realtime

import (
	"context"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// CursorValidator は、client が提示した cursor から replay を始められるかを判定します。
type CursorValidator interface {
	// Validate は、cursor より後ろを欠けなく replay できるなら nil を返します。replay floor（ADR-0072 —
	// 保存せず導出する）より前なら ErrCursorExpired、EventLog が読めなければ apperror.ErrUnavailable
	// （retry 可能）を返します。cursor 0 は初期位置（stream の先頭から）です。
	Validate(ctx context.Context, streamID rt.StreamID, cursor rt.Sequence) error
}

type cursorValidator struct {
	log    rt.EventLogStore
	clock  clock.Clock
	tracer observability.LayerTracer
}

// NewCursorValidator は、CursorValidator を生成します。
func NewCursorValidator(log rt.EventLogStore, clk clock.Clock, tf observability.TracerFactory) CursorValidator {
	return &cursorValidator{log: log, clock: clk, tracer: tf.Usecase()}
}

// Validate は、replay floor を EventLog の状態から導出します。cursor より後ろに現存する最初の event を
// 1 回の強い一貫性の読み取りで取り、それが cursor+1 なら保持期間内かどうか、cursor+1 より後ろなら gap、
// 無ければ（初期位置でない限り）cursor 自身の event が残っているかを見ます。「cursor+1 の有無」と
// 「後ろの event の有無」を別々に読むと、その間に relay が cursor+1 を append しただけで gap に見えるため、
// 1 回の読み取りにまとめます。
func (v *cursorValidator) Validate(ctx context.Context, streamID rt.StreamID, cursor rt.Sequence) error {
	ctx, endSpan := v.tracer.Start(ctx)
	defer endSpan()

	res, err := v.log.ReadAfter(ctx, rt.ReadAfterQuery{StreamID: streamID, After: cursor, Limit: 1})
	if err != nil {
		return err
	}

	if len(res.Events) > 0 {
		next := res.Events[0]
		if next.Sequence != cursor+1 {
			return xerrors.Wrap(ErrCursorExpired, "the next event is gone while a later one exists")
		}

		if v.clock.Now().Sub(next.OccurredAt) > rt.EventLogRetention {
			return xerrors.Wrap(ErrCursorExpired, "the next event is older than the retention")
		}

		return nil
	}

	if cursor == 0 {
		return nil
	}

	_, ok, err := v.log.Find(ctx, streamID, cursor)
	if err != nil {
		return err
	}

	if !ok {
		return xerrors.Wrap(ErrCursorExpired, "the event at the cursor is gone")
	}

	return nil
}
