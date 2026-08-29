package realtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	mock_ctrlrealtime "go-boilerplate/internal/controller/realtime/mock"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

// sinkRecord は、mock の受け口が受けた呼び出しの記録です。
type sinkRecord struct {
	mu          sync.Mutex
	wakeups     map[rt.StreamID][]rt.Sequence
	revocations []rt.Revocation
}

// engineMocks は、engine の依存の mock と、受け口の記録、観測したログを数える関数です。
type engineMocks struct {
	sub     *mock_realtime.MockInstanceSubscription
	sleeper *mock_clock.MockSleeper
	waker   *mock_ctrlrealtime.MockWaker
	revoker *mock_ctrlrealtime.MockRevoker
	sinks   *sinkRecord
	// logCount は、msg のログの件数を返します（msg が空なら全件）。
	logCount func(msg string) int
}

func newEngine(t *testing.T, set Settings) (*Engine, engineMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := engineMocks{
		sub:     mock_realtime.NewMockInstanceSubscription(ctrl),
		sleeper: mock_clock.NewMockSleeper(ctrl),
		waker:   mock_ctrlrealtime.NewMockWaker(ctrl),
		revoker: mock_ctrlrealtime.NewMockRevoker(ctrl),
		sinks:   &sinkRecord{wakeups: map[rt.StreamID][]rt.Sequence{}},
	}
	m.waker.EXPECT().Wake(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, streamID rt.StreamID, upTo rt.Sequence) {
			m.sinks.mu.Lock()
			defer m.sinks.mu.Unlock()

			m.sinks.wakeups[streamID] = append(m.sinks.wakeups[streamID], upTo)
		}).AnyTimes()
	m.revoker.EXPECT().Revoke(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, subject string, destination rt.StreamID) {
			m.sinks.mu.Lock()
			defer m.sinks.mu.Unlock()

			m.sinks.revocations = append(m.sinks.revocations, rt.Revocation{Subject: subject, Destination: destination})
		}).AnyTimes()

	log, logs := logging.NewObservedTestLogger(t)
	m.logCount = func(msg string) int {
		if msg == "" {
			return logs.Len()
		}

		return logs.FilterMessage(msg).Len()
	}

	return NewEngine(m.sub, m.waker, m.revoker, m.sleeper, log, observability.NewNoopTracerFactory(t), set), m
}

func wakeup(streamID rt.StreamID, seq rt.Sequence) rt.Notification {
	return rt.Notification{
		Kind:    rt.KindWakeup,
		Wakeup:  rt.Wakeup{EventID: "e", StreamID: streamID, Sequence: seq},
		Receipt: "r-" + string(streamID),
	}
}

func revocation(subject string, destination rt.StreamID) rt.Notification {
	return rt.Notification{
		Kind:       rt.KindRevocation,
		Revocation: rt.Revocation{Subject: subject, Destination: destination},
		Receipt:    "r-" + subject,
	}
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

		t.Run("負の設定値も既定値に寄せる", func(t *testing.T) {
			t.Parallel()

			e, _ := newEngine(t, Settings{BatchSize: -1, ErrorBackoff: -time.Second})
			assert.Equal(t, Settings{BatchSize: DefaultBatchSize, ErrorBackoff: DefaultErrorBackoff}, e.set)
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
				m.sub.EXPECT().
					Receive(gomock.Any(), DefaultBatchSize).
					DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
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
				m.sub.EXPECT().
					Receive(gomock.Any(), gomock.Any()).
					DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
						cancel()

						return nil, nil
					}),
			)

			require.NoError(t, e.Run(ctx))
			assert.Equal(t, 1, m.logCount("failed to receive realtime notifications"))
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
			m.sub.EXPECT().
				Receive(gomock.Any(), gomock.Any()).
				DoAndReturn(func(context.Context, int) ([]rt.Notification, error) {
					cancel()

					return nil, apperror.ErrCanceled
				})

			require.NoError(t, e.Run(ctx))
			assert.Equal(t, 0, m.logCount(""))
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

			assert.Equal(t, 1, m.logCount("discarding realtime notifications of unknown kind"))
			assert.Empty(t, m.sinks.wakeups)
		})

		t.Run("空の受信は何もしない", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			e.dispatch(t.Context(), e.logging, nil)

			assert.Equal(t, 0, m.logCount(""))
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
			assert.Equal(t, 1, m.logCount("failed to delete realtime notification"))
		})

		t.Run("停止中の削除失敗は記録しない", func(t *testing.T) {
			t.Parallel()

			e, m := newEngine(t, Settings{})
			batch := []rt.Notification{wakeup("s1", 1)}
			ctx, cancel := context.WithCancel(t.Context())
			m.sub.EXPECT().Delete(gomock.Any(), batch[0]).DoAndReturn(func(context.Context, rt.Notification) error {
				cancel()

				return apperror.ErrCanceled
			})

			e.dispatch(ctx, e.logging, batch)

			assert.Len(t, m.sinks.wakeups, 1)
			assert.Equal(t, 0, m.logCount("failed to delete realtime notification"))
		})
	})
}

func Test_coalesce(t *testing.T) {
	t.Parallel()

	wakeups, revocations, unknown := coalesce([]rt.Notification{
		wakeup("s1", 3), wakeup("s2", 1), wakeup("s1", 9), wakeup("s1", 4),
		revocation("u1", "s1"), revocation("u2", "s2"),
		{Receipt: "r-unknown"},
		{Kind: "other", Receipt: "r-other"},
	})

	assert.Equal(t, map[rt.StreamID]rt.Sequence{"s1": 9, "s2": 1}, wakeups, "stream ごとに最大の sequence")
	assert.Equal(
		t,
		[]rt.Revocation{{Subject: "u1", Destination: "s1"}, {Subject: "u2", Destination: "s2"}},
		revocations,
	)
	assert.Equal(t, 2, unknown)
}
