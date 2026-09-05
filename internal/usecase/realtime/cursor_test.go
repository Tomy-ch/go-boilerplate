package realtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
	"go-boilerplate/pkg/xerrors"
)

var (
	now         = time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	errStoreOff = xerrors.Wrap(apperror.ErrUnavailable, "store off")
)

func eventAt(seq rt.Sequence, occurredAt time.Time) rt.DeliveryEvent {
	return rt.DeliveryEvent{
		EventID:       "evt-" + seq.String(),
		StreamID:      "s",
		Sequence:      seq,
		Type:          "t",
		OccurredAt:    occurredAt,
		SchemaVersion: 1,
	}
}

func newValidator(t *testing.T) (*cursorValidator, *mock_realtime.MockEventLogStore) {
	t.Helper()

	ctrl := gomock.NewController(t)
	log := mock_realtime.NewMockEventLogStore(ctrl)
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()

	return &cursorValidator{log: log, clock: clk, tracer: observability.NewNoopTracerFactory(t).Usecase()}, log
}

func TestNewCursorValidator(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	v := NewCursorValidator(
		mock_realtime.NewMockEventLogStore(ctrl),
		mock_clock.NewMockClock(ctrl),
		observability.NewNoopTracerFactory(t),
	)
	assert.NotNil(t, v)
}

func readAfter(after rt.Sequence) rt.ReadAfterQuery {
	return rt.ReadAfterQuery{StreamID: "s", After: after, Limit: 1}
}

func Test_cursorValidator_Validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursor+1 が保持期間内にあれば replay できる", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(4, now.Add(-time.Hour))}}, nil)

			require.NoError(t, v.Validate(t.Context(), "s", 3))
		})

		t.Run("cursor+1 がちょうど保持期間の境界なら replay できる", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(4, now.Add(-rt.EventLogRetention))}}, nil)

			require.NoError(t, v.Validate(t.Context(), "s", 3))
		})

		t.Run("cursor が stream の先頭（最新）にあれば replay するものが無く成功する", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(3)).Return(eventAt(3, now), true, nil)

			require.NoError(t, v.Validate(t.Context(), "s", 3))
		})

		t.Run("追記済みの位置より先の cursor は失効ではない（relay がまだ書いていない）", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(3)).Return(rt.DeliveryEvent{}, false, nil)
			log.EXPECT().AppendedThrough(gomock.Any(), rt.StreamID("s")).Return(rt.Sequence(2), nil)

			require.NoError(t, v.Validate(t.Context(), "s", 3))
		})

		t.Run("1 度も追記していない stream への cursor も失効ではない", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(1)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(1)).Return(rt.DeliveryEvent{}, false, nil)
			log.EXPECT().AppendedThrough(gomock.Any(), rt.StreamID("s")).Return(rt.Sequence(0), nil)

			require.NoError(t, v.Validate(t.Context(), "s", 1))
		})

		t.Run("初期位置で stream が空なら成功する", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(0)).Return(rt.ReadAfterResult{}, nil)

			require.NoError(t, v.Validate(t.Context(), "s", 0))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursor+1 が保持期間より古ければ ErrCursorExpired", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(4, now.Add(-rt.EventLogRetention-time.Second))}}, nil)

			require.ErrorIs(t, v.Validate(t.Context(), "s", 3), ErrCursorExpired)
		})

		t.Run("cursor より後ろに現存する最初の event が cursor+1 でなければ（gap）ErrCursorExpired", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(9, now)}}, nil)

			require.ErrorIs(t, v.Validate(t.Context(), "s", 3), ErrCursorExpired)
		})

		t.Run("初期位置で先頭の event が消えていれば ErrCursorExpired", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(0)).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(5, now)}}, nil)

			require.ErrorIs(t, v.Validate(t.Context(), "s", 0), ErrCursorExpired)
		})

		t.Run("追記済みの位置以下なのに cursor の event が無ければ（消えた）ErrCursorExpired", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(3)).Return(rt.DeliveryEvent{}, false, nil)
			log.EXPECT().AppendedThrough(gomock.Any(), rt.StreamID("s")).Return(rt.Sequence(9), nil)

			require.ErrorIs(t, v.Validate(t.Context(), "s", 3), ErrCursorExpired)
		})

		t.Run("追記済みの位置が読めなければ store のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(3)).Return(rt.DeliveryEvent{}, false, nil)
			log.EXPECT().AppendedThrough(gomock.Any(), rt.StreamID("s")).Return(rt.Sequence(0), errStoreOff)

			err := v.Validate(t.Context(), "s", 3)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, ErrCursorExpired)
		})

		t.Run("EventLog が読めなければ store のエラー（ErrUnavailable）をそのまま返す", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, errStoreOff)

			err := v.Validate(t.Context(), "s", 3)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, ErrCursorExpired)
		})

		t.Run("cursor 自身の読み取りに失敗すれば store のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			v, log := newValidator(t)
			log.EXPECT().ReadAfter(gomock.Any(), readAfter(3)).Return(rt.ReadAfterResult{}, nil)
			log.EXPECT().
				Find(gomock.Any(), rt.StreamID("s"), rt.Sequence(3)).
				Return(rt.DeliveryEvent{}, false, errStoreOff)

			require.ErrorIs(t, v.Validate(t.Context(), "s", 3), apperror.ErrUnavailable)
		})
	})
}
