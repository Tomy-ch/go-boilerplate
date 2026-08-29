package realtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

func newReplayer(t *testing.T) (*replayer, *mock_realtime.MockEventLogStore) {
	t.Helper()

	log := mock_realtime.NewMockEventLogStore(gomock.NewController(t))

	return &replayer{log: log, tracer: observability.NewNoopTracerFactory(t).Usecase()}, log
}

func TestNewReplayer(t *testing.T) {
	t.Parallel()

	r := NewReplayer(
		mock_realtime.NewMockEventLogStore(gomock.NewController(t)),
		observability.NewNoopTracerFactory(t),
	)

	assert.NotNil(t, r)
}

func Test_replayer_ReadPage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursor より後ろを ReplayPageLimit 件までで読む", func(t *testing.T) {
			t.Parallel()

			r, log := newReplayer(t)
			log.EXPECT().
				ReadAfter(gomock.Any(), rt.ReadAfterQuery{StreamID: "s", After: 7, Limit: ReplayPageLimit}).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(8, now)}}, nil)

			events, hasMore, err := r.ReadPage(context.Background(), "s", 7)

			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, rt.Sequence(8), events[0].Sequence)
			assert.False(t, hasMore)
		})

		t.Run("store が打ち切ったら hasMore を伝える", func(t *testing.T) {
			t.Parallel()

			r, log := newReplayer(t)
			log.EXPECT().ReadAfter(gomock.Any(), gomock.Any()).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(1, now)}, HasMore: true}, nil)

			_, hasMore, err := r.ReadPage(context.Background(), "s", 0)

			require.NoError(t, err)
			assert.True(t, hasMore)
		})

		t.Run("飛び番でもそのまま返し gap の判定はしない", func(t *testing.T) {
			t.Parallel()

			r, log := newReplayer(t)
			log.EXPECT().ReadAfter(gomock.Any(), gomock.Any()).
				Return(rt.ReadAfterResult{Events: []rt.DeliveryEvent{eventAt(9, now)}}, nil)

			events, _, err := r.ReadPage(context.Background(), "s", 3)

			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, rt.Sequence(9), events[0].Sequence)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("store が読めなければ ErrUnavailable をそのまま返す", func(t *testing.T) {
			t.Parallel()

			r, log := newReplayer(t)
			log.EXPECT().ReadAfter(gomock.Any(), gomock.Any()).Return(rt.ReadAfterResult{}, errStoreOff)

			events, hasMore, err := r.ReadPage(context.Background(), "s", 0)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, events)
			assert.False(t, hasMore)
		})
	})
}
