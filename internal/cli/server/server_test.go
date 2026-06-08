package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServeCommand(t *testing.T) {
	t.Parallel()

	cmd := NewServeCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestRunServer(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	shutdownTimeout := appCfg.ShutdownTimeout()

	t.Run("正常系_シグナル受信後に起動しグレースフルシャットダウンが実行される", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		var startCalled bool
		//nolint:unparam // runServer のシグネチャ func(context.Context) error に合わせる必要がある
		start := func(context.Context) error {
			startCalled = true
			return nil
		}

		var (
			stopMetricsCalled bool
			stopMetricsCtx    context.Context
		)
		stopMetrics := func(c context.Context) {
			stopMetricsCalled = true
			stopMetricsCtx = c
		}

		var (
			stopDeadline    time.Time
			stopHasDeadline bool
			stopCtx         context.Context
		)
		//nolint:unparam // runServer のシグネチャ func(context.Context) error に合わせる必要がある
		stop := func(c context.Context) error {
			stopCtx = c
			stopDeadline, stopHasDeadline = c.Deadline()
			return nil
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- runServer(ctx, shutdownTimeout, start, stop, stopMetrics)
		}()

		// 起動から一定時間「稼働」させてからシグナル受信相当のキャンセルを行う。
		// 停止タイムアウトが「停止開始時点」から計測されること（起動時点ではない）を検証するため、
		// 稼働時間（120ms）を許容誤差（60ms）より十分大きく取る。
		time.Sleep(120 * time.Millisecond)
		stopStart := time.Now()
		cancel()

		require.NoError(t, <-errCh)

		assert.True(t, startCalled, "startApp が呼ばれること")
		assert.True(t, stopMetricsCalled, "stopMetrics が呼ばれること")
		require.True(t, stopHasDeadline, "stopApp は期限付き context を受け取ること")
		// 停止用 context の deadline が「停止開始時点 + ShutdownTimeout」付近であること
		// （= 稼働時間に消費されていないこと。b0a8e21 の回帰防止）。
		assert.WithinDuration(t, stopStart.Add(shutdownTimeout), stopDeadline, 60*time.Millisecond)
		// メトリクス停止とアプリ停止は同一の停止用 context を共有すること。
		assert.Equal(t, stopCtx, stopMetricsCtx)
	})

	t.Run("正常系_stopMetricsがnilでもパニックせず停止する", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 即時にシグナル受信相当にする

		start := func(context.Context) error { return nil }
		stop := func(context.Context) error { return nil }

		require.NoError(t, runServer(ctx, shutdownTimeout, start, stop, nil))
	})

	t.Run("異常系_起動失敗時は停止処理を行わずエラーを返す", func(t *testing.T) {
		t.Parallel()

		startErr := errors.New("start failed")
		start := func(context.Context) error { return startErr }

		var stopCalled bool
		stop := func(context.Context) error {
			stopCalled = true
			return nil
		}

		err := runServer(context.Background(), shutdownTimeout, start, stop, nil)
		require.ErrorIs(t, err, startErr)
		assert.False(t, stopCalled, "起動失敗時は stopApp を呼ばないこと")
	})
}
