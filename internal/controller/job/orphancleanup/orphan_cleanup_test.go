package orphancleanup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_realtime "go-boilerplate/internal/usecase/realtime/mock"
)

// newSweeperMock は、掃除役の mock と、それを返すファクトリを組みます。
func newSweeperMock(t *testing.T) (*mock_realtime.MockOrphanSweeper, SweeperFactory) {
	t.Helper()

	sweeper := mock_realtime.NewMockOrphanSweeper(gomock.NewController(t))

	return sweeper, func(context.Context) (ucrealtime.OrphanSweeper, error) { return sweeper, nil }
}

// newJob は、観測用 Logger を差した job と、その掃除役の mock を返します。
func newJob(t *testing.T, log logging.Logger) (*jobImpl, *mock_realtime.MockOrphanSweeper) {
	t.Helper()

	sweeper, factory := newSweeperMock(t)
	j, ok := New(log, observability.NewNoopTracerFactory(t), factory).(*jobImpl)
	require.True(t, ok)

	return j, sweeper
}

func TestNew(t *testing.T) {
	t.Parallel()

	j, _ := newJob(t, logging.NewTestLogger(t))
	assert.NotNil(t, j)
}

func Test_jobImpl_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, jobName, (&jobImpl{}).Name())
}

func Test_jobImpl_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("掃除を 1 度だけ実行し、内訳を出力する", func(t *testing.T) {
			t.Parallel()

			observedLog, logs := logging.NewObservedTestLogger(t)
			j, sweeper := newJob(t, observedLog)
			sweeper.EXPECT().Sweep(gomock.Any()).Return(ucrealtime.SweepResult{Detected: 2, Reclaimed: 2}, nil)

			require.NoError(t, j.Execute(t.Context(), nil))

			entries := logs.FilterMessage(resultMessage).All()
			require.Len(t, entries, 1)
			assert.Equal(t, "info", entries[0].Level.String())
			assert.Equal(t, int64(2), entries[0].ContextMap()[logging.JobResultKey])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引数を受け取ったら掃除せずに止まる", func(t *testing.T) {
			t.Parallel()

			j, _ := newJob(t, logging.NewTestLogger(t))

			require.ErrorIs(t, j.Execute(t.Context(), []string{"--unknown"}), errUnknownFlag)
		})

		t.Run("掃除役を組めなければ掃除せずに返す", func(t *testing.T) {
			t.Parallel()

			// Realtime を配線していない環境ではここで初めて失敗し、他のジョブは影響を受けない。
			factory := func(context.Context) (ucrealtime.OrphanSweeper, error) { return nil, apperror.ErrInvalidArgument }
			j, ok := New(logging.NewTestLogger(t), observability.NewNoopTracerFactory(t), factory).(*jobImpl)
			require.True(t, ok)

			require.ErrorIs(t, j.Execute(t.Context(), nil), apperror.ErrInvalidArgument)
		})

		t.Run("掃除が失敗しても、確定した内訳を出力してからエラーを返す", func(t *testing.T) {
			t.Parallel()

			observedLog, logs := logging.NewObservedTestLogger(t)
			j, sweeper := newJob(t, observedLog)
			sweeper.EXPECT().Sweep(gomock.Any()).
				Return(ucrealtime.SweepResult{Detected: 2, Reclaimed: 1, Failed: 1}, apperror.ErrUnavailable)

			require.ErrorIs(t, j.Execute(t.Context(), nil), apperror.ErrUnavailable)

			entries := logs.FilterMessage(abortedMessage).All()
			require.Len(t, entries, 1)
			assert.Equal(t, "warn", entries[0].Level.String())
			assert.Equal(t, int64(1), entries[0].ContextMap()[logging.JobResultKey])
		})
	})
}

func Test_resultFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検出・回収・見送りをジョブ共通のキーへ写す", func(t *testing.T) {
			t.Parallel()

			log, logs := logging.NewObservedTestLogger(t)
			log.Info(t.Context(), "result",
				resultFields(ucrealtime.SweepResult{Detected: 5, Claimed: 4, Reclaimed: 3, Skipped: 2, Failed: 1})...)

			entries := logs.All()
			require.Len(t, entries, 1)
			// 引き受け数と失敗数は載せない。失敗は返り値のエラーの chain が運ぶ。
			assert.Equal(t, map[string]any{
				logging.JobScannedKey: int64(5),
				logging.JobResultKey:  int64(3),
				logging.JobSkippedKey: int64(2),
			}, entries[0].ContextMap())
		})
	})
}
