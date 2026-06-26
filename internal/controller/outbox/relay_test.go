package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	outboxctrl "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testBatchSize    = int32(100)
	testPollInterval = time.Second
	testErrorBackoff = 5 * time.Second
)

func newEngine(t *testing.T, uc *mock_relay.MockRelayUsecase, sleeper *mock_clock.MockSleeper) *outboxctrl.Engine {
	t.Helper()
	return outboxctrl.NewEngine(uc, sleeper, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t),
		outboxctrl.Settings{BatchSize: testBatchSize, PollInterval: testPollInterval, ErrorBackoff: testErrorBackoff})
}

func TestEngine_Run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending を捌き切ったら PollInterval で待機し ctx 完了で停止する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper := mock_clock.NewMockSleeper(ctrl)

			uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(outboxuc.RelayResult{}, nil)
			sleeper.EXPECT().Sleep(gomock.Any(), testPollInterval).Return(context.Canceled)

			require.NoError(t, newEngine(t, uc, sleeper).Run(context.Background()))
		})

		t.Run("満杯かつ publish 進捗ありの間は待機せず連続消化する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper := mock_clock.NewMockSleeper(ctrl)

			gomock.InOrder(
				uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(
					outboxuc.RelayResult{Claimed: int(testBatchSize), Published: int(testBatchSize)}, nil),
				uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(outboxuc.RelayResult{}, nil),
			)
			// 満杯かつ進捗ありの回は Sleep されず、捌き切った回だけ PollInterval で待機する。
			sleeper.EXPECT().Sleep(gomock.Any(), testPollInterval).Return(context.Canceled).Times(1)

			require.NoError(t, newEngine(t, uc, sleeper).Run(context.Background()))
		})

		t.Run("満杯でも publish 進捗が無ければ PollInterval で待機しホットループしない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper := mock_clock.NewMockSleeper(ctrl)

			// 全件 publish 失敗（Published==0）の満杯バッチは、待機ゼロで再 claim せず PollInterval を挟む。
			uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(
				outboxuc.RelayResult{Claimed: int(testBatchSize), Published: 0}, nil)
			sleeper.EXPECT().Sleep(gomock.Any(), testPollInterval).Return(context.Canceled).Times(1)

			require.NoError(t, newEngine(t, uc, sleeper).Run(context.Background()))
		})

		t.Run("RelayBatch エラー時は ErrorBackoff で待機する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			// エラー時は lag を記録しない（呼べば未期待呼び出しで失敗する）。
			sleeper := mock_clock.NewMockSleeper(ctrl)

			uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(
				outboxuc.RelayResult{}, errors.New("batch failed"))
			sleeper.EXPECT().Sleep(gomock.Any(), testErrorBackoff).Return(context.Canceled)

			require.NoError(t, newEngine(t, uc, sleeper).Run(context.Background()))
		})

		t.Run("RecordLag が失敗してもログのみでループを継続する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			sleeper := mock_clock.NewMockSleeper(ctrl)

			uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).Return(outboxuc.RelayResult{}, nil).AnyTimes()
			uc.EXPECT().RecordLag(gomock.Any()).Return(errors.New("lag failed")).AnyTimes()
			sleeper.EXPECT().Sleep(gomock.Any(), testPollInterval).Return(context.Canceled)

			require.NoError(t, newEngine(t, uc, sleeper).Run(context.Background()))
		})

		t.Run("開始時に ctx が完了済みなら RelayBatch を呼ばず停止する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper := mock_clock.NewMockSleeper(ctrl)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			require.NoError(t, newEngine(t, uc, sleeper).Run(ctx))
		})

		t.Run("RelayBatch エラー後に ctx が完了していれば待機せず停止する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).AnyTimes()
			sleeper := mock_clock.NewMockSleeper(ctrl)

			ctx, cancel := context.WithCancel(context.Background())
			uc.EXPECT().RelayBatch(gomock.Any(), testBatchSize).DoAndReturn(
				func(_ context.Context, _ int32) (outboxuc.RelayResult, error) {
					cancel()
					return outboxuc.RelayResult{}, errors.New("batch failed")
				})

			// ctx 完了済みのため Sleep は呼ばれない。
			require.NoError(t, newEngine(t, uc, sleeper).Run(ctx))
			assert.Equal(t, context.Canceled, ctx.Err())
		})
	})
}
