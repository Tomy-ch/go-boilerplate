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

// Validate は、replay floor を EventLog の状態から導出します。判定は 3 つで、順に見ます:
// cursor+1 の item があればそれが保持期間内かどうか、無ければ後ろに item があるか（あれば gap）、
// 最後に cursor 自身の item が残っているか（初期位置でなければ必要）。
func (v *cursorValidator) Validate(ctx context.Context, streamID rt.StreamID, cursor rt.Sequence) error {
	ctx, endSpan := v.tracer.Start(ctx)
	defer endSpan()

	next, ok, err := v.log.Find(ctx, streamID, cursor+1)
	if err != nil {
		return err
	}

	if ok {
		if v.clock.Now().Sub(next.OccurredAt) > rt.EventLogRetention {
			return xerrors.Wrap(ErrCursorExpired, "the next event is older than the retention")
		}

		return nil
	}

	latest, hasLatest, err := v.log.Latest(ctx, streamID)
	if err != nil {
		return err
	}

	if hasLatest && latest.Sequence > cursor {
		return xerrors.Wrap(ErrCursorExpired, "the next event is gone while a later one exists")
	}

	if cursor == 0 {
		return nil
	}

	_, ok, err = v.log.Find(ctx, streamID, cursor)
	if err != nil {
		return err
	}

	if !ok {
		return xerrors.Wrap(ErrCursorExpired, "the event at the cursor is gone")
	}

	return nil
}
