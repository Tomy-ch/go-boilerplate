package outbox

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	mock_relay "go-boilerplate/internal/usecase/outbox/mock"
	"go-boilerplate/pkg/xerrors"

	"go.uber.org/mock/gomock"
)

func newObserveLagEngine(t *testing.T, uc *mock_relay.MockRelayUsecase) *Engine {
	t.Helper()
	ctrl := gomock.NewController(t)
	sleeper := mock_clock.NewMockSleeper(ctrl)
	return NewEngine(uc, sleeper, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t),
		Settings{BatchSize: 100, PollInterval: time.Second, ErrorBackoff: 5 * time.Second})
}

func TestEngine_observeLag(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx有効時はRecordLagとRecordBlockedStreamsを呼ぶ", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(nil).Times(1)
			uc.EXPECT().RecordBlockedStreams(gomock.Any()).Return(nil).Times(1)

			e := newObserveLagEngine(t, uc)
			e.observeLag(context.Background(), logging.NewTestLogger(t))
		})

		t.Run("記録が失敗してもpanicせずログのみで戻る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			uc.EXPECT().RecordLag(gomock.Any()).Return(xerrors.New("lag failed")).Times(1)
			uc.EXPECT().RecordBlockedStreams(gomock.Any()).Return(xerrors.New("count failed")).Times(1)

			e := newObserveLagEngine(t, uc)
			e.observeLag(context.Background(), logging.NewTestLogger(t))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx完了時はいずれの記録も呼ばずスキップする", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock_relay.NewMockRelayUsecase(ctrl)
			// RecordLag / RecordBlockedStreams を EXPECT しない: 呼ばれれば未期待呼び出しで失敗する。

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			e := newObserveLagEngine(t, uc)
			e.observeLag(ctx, logging.NewTestLogger(t))
		})
	})
}
