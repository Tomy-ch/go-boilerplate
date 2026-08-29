package realtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest/observer"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

// recordingSinks は、sink への呼び出しを記録する test double です。
type recordingSinks struct {
	mu          sync.Mutex
	wakeups     map[rt.StreamID][]rt.Sequence
	revocations []rt.Revocation
}

func newRecordingSinks() *recordingSinks {
	return &recordingSinks{wakeups: map[rt.StreamID][]rt.Sequence{}}
}

func (s *recordingSinks) Wake(_ context.Context, streamID rt.StreamID, upTo rt.Sequence) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.wakeups[streamID] = append(s.wakeups[streamID], upTo)
}

func (s *recordingSinks) Revoke(_ context.Context, subject string, destination rt.StreamID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.revocations = append(s.revocations, rt.Revocation{Subject: subject, Destination: destination})
}

type engineMocks struct {
	sub     *mock_realtime.MockInstanceSubscription
	sleeper *mock_clock.MockSleeper
	sinks   *recordingSinks
	logs    *observer.ObservedLogs
}

func newEngine(t *testing.T, set Settings) (*Engine, engineMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := engineMocks{
		sub: mock_realtime.NewMockInstanceSubscription(ctrl), sleeper: mock_clock.NewMockSleeper(ctrl), sinks: newRecordingSinks(),
	}

	log, logs := logging.NewObservedTestLogger(t)
	m.logs = logs

	return NewEngine(m.sub, m.sinks, m.sinks, m.sleeper, log, observability.NewNoopTracerFactory(t), set), m
}

func wakeup(streamID rt.StreamID, seq rt.Sequence) rt.Notification {
	return rt.Notification{Kind: rt.KindWakeup, Wakeup: rt.Wakeup{EventID: "e", StreamID: streamID, Sequence: seq}, Receipt: "r-" + string(streamID)}
}

func revocation(subject string, destination rt.StreamID) rt.Notification {
	return rt.Notification{Kind: rt.KindRevocation, Revocation: rt.Revocation{Subject: subject, Destination: destination}, Receipt: "r-" + subject}
}

func TestNewEngine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値の設定は既定値に寄せる", func(t *testing.T) {
			t.Parallel()

			e, _ := newEngine(t, Settings{})
			assert.Equal(t, Settings{BatchSize: DefaultBatchSize, ErrorBackoff: DefaultErrorBackoff}, e.set)
		})

		t.Run("指定した設定はそのまま使う", func(t *testing.T) {
			t.Parallel()

			e, _ := newEngine(t, Settings{BatchSize: 3, ErrorBackoff: time.Second})
			assert.Equal(t, Settings{BatchSize: 3, ErrorBackoff: time.Second}, e.set)
		})
	})
}

func TestEngine_Run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受けた通知を sink へ渡して削除し、ctx 完了で nil を返す", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{BatchSize: 4})
			ctx, cancel := context.WithCancel(t.Context())
			batch := []rt.Notification{wakeup("s1", 3), wakeup("s1", 5), revocation("u1", "s1")}
			gomock.InOrder(
				m.sub.EXPECT().Receive(gomock.Any(), 4).Return(batch, nil),
				m.sub.EXPECT().Delete(gomock.Any(), batch[0]).Return(nil),
				m.sub.EXPECT().Delete(gomock.Any(), batch[1]).Return(nil),
				m.sub.EXPECT().Delete(gomock.Any(), batch[2]).DoAndReturn(func(context.Context, rt.Notification) error {
					cancel()

					return nil
				}),
			)

			require.NoError(t, e.Run(ctx))
			assert.Equal(t, map[rt.StreamID][]rt.Sequence{"s1": {5}}, m.sinks.wakeups, "同じ stream は最大の sequence 1 件に畳む")
			assert.Equal(t, []rt.Revocation{{Subject: "u1", Destination: "s1"}}, m.sinks.revocations)
		})

		t.Run("開始時点で ctx が完了していれば受信せずに nil を返す", func(t *testing.T) {
			t.Parallel()

			e, _ := newEngine(t, Settings{})
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			require.NoError(t, e.Run(ctx))
		})

		t.Run("空の受信は sink を呼ばず次の受信へ進む", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			ctx, cancel := context.WithCancel(t.Context())
			gomock.InOrder(
				m.sub.EXPECT().Receive(gomock.Any(), DefaultBatchSize).Return(nil, nil),
				m.sub.EXPECT().Receive(gomock.Any(), DefaultBatchSize).DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
					cancel()

					return nil, nil
				}),
			)

			require.NoError(t, e.Run(ctx))
			assert.Empty(t, m.sinks.wakeups)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信に失敗したら記録して ErrorBackoff 待ち、続ける", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{ErrorBackoff: 7 * time.Second})
			ctx, cancel := context.WithCancel(t.Context())
			gomock.InOrder(
				m.sub.EXPECT().Receive(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrUnavailable),
				m.sleeper.EXPECT().Sleep(gomock.Any(), 7*time.Second).Return(nil),
				m.sub.EXPECT().Receive(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
					cancel()

					return nil, nil
				}),
			)

			require.NoError(t, e.Run(ctx))
			assert.Equal(t, 1, m.logs.FilterMessage("failed to receive realtime notifications").Len())
		})

		t.Run("backoff 中に ctx が完了すれば nil を返す", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			m.sub.EXPECT().Receive(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrUnavailable)
			m.sleeper.EXPECT().Sleep(gomock.Any(), DefaultErrorBackoff).Return(context.Canceled)

			require.NoError(t, e.Run(t.Context()))
		})

		t.Run("ctx 完了で受信が失敗しても記録せずに nil を返す", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			ctx, cancel := context.WithCancel(t.Context())
			m.sub.EXPECT().Receive(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
				cancel()

				return nil, apperror.ErrCanceled
			})

			require.NoError(t, e.Run(ctx))
			assert.Equal(t, 0, m.logs.Len())
		})
	})
}

func TestEngine_dispatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("種別の無い通知は警告して削除する", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			unknown := rt.Notification{Receipt: "r-x"}
			m.sub.EXPECT().Delete(gomock.Any(), unknown).Return(nil)

			e.dispatch(t.Context(), e.logging, []rt.Notification{unknown})

			assert.Equal(t, 1, m.logs.FilterMessage("discarding realtime notifications of unknown kind").Len())
			assert.Empty(t, m.sinks.wakeups)
		})

		t.Run("空の受信は何もしない", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			e.dispatch(t.Context(), e.logging, nil)

			assert.Equal(t, 0, m.logs.Len())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除に失敗しても sink へは渡し済みで、失敗を記録して続ける", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			batch := []rt.Notification{wakeup("s1", 1), wakeup("s2", 2)}
			m.sub.EXPECT().Delete(gomock.Any(), batch[0]).Return(apperror.ErrUnavailable)
			m.sub.EXPECT().Delete(gomock.Any(), batch[1]).Return(nil)

			e.dispatch(t.Context(), e.logging, batch)

			assert.Len(t, m.sinks.wakeups, 2)
			assert.Equal(t, 1, m.logs.FilterMessage("failed to delete realtime notification").Len())
		})
	})
}

func Test_coalesce(t *testing.T) {
	t.Parallel()

	wakeups, revocations, unknown := coalesce([]rt.Notification{
		wakeup("s1", 3), wakeup("s2", 1), wakeup("s1", 9), wakeup("s1", 4),
		revocation("u1", "s1"), revocation("u2", "s2"),
		{Receipt: "r-unknown"}, {Kind: "other", Receipt: "r-other"},
	})

	assert.Equal(t, map[rt.StreamID]rt.Sequence{"s1": 9, "s2": 1}, wakeups, "stream ごとに最大の sequence")
	assert.Equal(t, []rt.Revocation{{Subject: "u1", Destination: "s1"}, {Subject: "u2", Destination: "s2"}}, revocations)
	assert.Equal(t, 2, unknown)
}
