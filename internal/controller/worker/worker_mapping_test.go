package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
	"go-boilerplate/pkg/backoff"
	"go-boilerplate/pkg/xerrors"
)

func Test_circuit_onSuccess(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Half-open での成功は Closed へ復帰し openCount を 0 に戻す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure() // Closed -> Open (threshold=1)
			c.toHalfOpen()
			require.Equal(t, phaseHalfOpen, c.phaseNow())

			c.onSuccess()

			assert.Equal(t, phaseClosed, c.phaseNow())
			assert.Zero(t, c.openCount)
		})

		t.Run("Closed での成功では状態を変えず失敗カウントのみリセットする", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(3, bo)
			c.onFailure() // failures=1, まだ Closed
			require.Equal(t, phaseClosed, c.phaseNow())

			c.onSuccess()

			assert.Equal(t, phaseClosed, c.phaseNow())
			assert.Zero(t, c.failures)
		})
	})
}

func Test_circuit_toHalfOpen(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Open からは Half-open へ遷移する", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure() // Closed -> Open
			require.Equal(t, phaseOpen, c.phaseNow())

			c.toHalfOpen()

			assert.Equal(t, phaseHalfOpen, c.phaseNow())
		})

		t.Run("Open 以外の状態では遷移しない", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			require.Equal(t, phaseClosed, c.phaseNow())

			c.toHalfOpen()

			assert.Equal(t, phaseClosed, c.phaseNow())
		})
	})
}

func Test_keyedDispatcher_dispatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PartitionKey が空の場合はそのまま proc を起動する", func(t *testing.T) {
			t.Parallel()

			got := make(chan string, 1)
			kd := newKeyedDispatcher(1, func(_ context.Context, m bw.Message) {
				got <- m.ID
			})

			kd.dispatch(context.Background(), bw.Message{ID: "a"})

			select {
			case id := <-got:
				assert.Equal(t, "a", id)
			case <-time.After(time.Second):
				t.Fatal("proc が起動されなかった")
			}
		})

		t.Run("PartitionKey が非空の場合は key ごとの runner 経由で proc を実行する", func(t *testing.T) {
			t.Parallel()

			got := make(chan string, 1)
			kd := newKeyedDispatcher(1, func(_ context.Context, m bw.Message) {
				got <- m.ID
			})

			kd.dispatch(context.Background(), bw.Message{ID: "a", PartitionKey: "k"})

			select {
			case id := <-got:
				assert.Equal(t, "a", id)
			case <-time.After(time.Second):
				t.Fatal("proc が起動されなかった")
			}
		})
	})
}

func Test_keyedDispatcher_runKey(t *testing.T) {
	t.Parallel()
	t.Skip("keyRunner の FIFO 直列処理は engine 統合テスト TestEngine_Run/engineRunPartitionKeySerialized でカバー")
}

func Test_run_waitCooldown(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		set := baseSettings()
		set.CircuitFailureThreshold = 1 // 1 度の失敗で Open へ遷移させる
		set.CircuitOpenBackoffInitial = 10 * time.Millisecond
		return newRun(newTestEngine(t, set, w), w)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cooldown 経過後に Half-open へ遷移して true を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			r.cb.onFailure() // Open へ遷移させ cooldown を確定
			require.Equal(t, phaseOpen, r.cb.phaseNow())

			ok := r.waitCooldown(context.Background())

			assert.True(t, ok)
			assert.Equal(t, phaseHalfOpen, r.cb.phaseNow())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待機中に ctx がキャンセルされると false を返し遷移しない", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			r.cb.onFailure()
			require.Equal(t, phaseOpen, r.cb.phaseNow())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			ok := r.waitCooldown(ctx)

			assert.False(t, ok)
			assert.Equal(t, phaseOpen, r.cb.phaseNow())
		})
	})
}

func Test_run_warnIfPoison(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T, threshold int) (*run, func() int) {
		t.Helper()
		logger, observed := logging.NewObservedTestLogger(t)
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		set := baseSettings()
		set.ReceiveCountWarnThreshold = threshold
		eng, err := New([]bw.Worker{w}, set, observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), logger)
		require.NoError(t, err)
		return newRun(eng, w), func() int { return observed.FilterMessage("receive count threshold reached").Len() }
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ReceiveCount が閾値以上の場合は warn ログを出す", func(t *testing.T) {
			t.Parallel()

			r, warnCount := newTestRun(t, 3)

			r.warnIfPoison(context.Background(), bw.Message{ID: "a", ReceiveCount: 3})

			assert.Equal(t, 1, warnCount())
		})

		t.Run("ReceiveCount が閾値未満の場合は warn しない", func(t *testing.T) {
			t.Parallel()

			r, warnCount := newTestRun(t, 3)

			r.warnIfPoison(context.Background(), bw.Message{ID: "a", ReceiveCount: 2})

			assert.Zero(t, warnCount())
		})

		t.Run("閾値が 0 以下の場合は warn しない", func(t *testing.T) {
			t.Parallel()

			r, warnCount := newTestRun(t, 0)

			r.warnIfPoison(context.Background(), bw.Message{ID: "a", ReceiveCount: 100})

			assert.Zero(t, warnCount())
		})
	})
}

func Test_run_triggerFatal(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		r := newRun(newTestEngine(t, baseSettings(), w), w)
		r.cancel = func() {} // loop 未起動でも triggerFatal が cancel を呼べるようにする
		return r
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回の Fatal を記録する", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			boom := xerrors.New("boom")

			r.triggerFatal(boom)

			require.ErrorIs(t, r.fatalErr(), boom)
		})

		t.Run("既に Fatal が記録済みの場合は上書きしない", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			first := xerrors.New("first")
			second := xerrors.New("second")

			r.triggerFatal(first)
			r.triggerFatal(second)

			require.ErrorIs(t, r.fatalErr(), first)
		})
	})
}

func Test_run_loop(t *testing.T) {
	t.Parallel()
	t.Skip("poll loop 本体は engine 統合テスト TestEngine_Run（ctx cancel/Ack/Nack 等）でカバー")
}

func Test_run_acquire(t *testing.T) {
	t.Parallel()
	t.Skip("Receive 許可の二段ゲートは engine 統合テスト TestEngine_Run/engineRunCircuitOpenAndRecover・engineRunAcquireInterruptedByCancel でカバー")
}

func Test_run_process(t *testing.T) {
	t.Parallel()
	t.Skip("1 メッセージ処理単位は engine 統合テスト TestEngine_Run/engineRunConcurrencyBounded 等でカバー")
}

func Test_run_finishMessage(t *testing.T) {
	t.Parallel()
	t.Skip("in-flight 解放と poll loop 起床は engine 統合テスト TestEngine_Run/engineRunMaxInFlightBounded でカバー")
}

func Test_run_safeHandle(t *testing.T) {
	t.Parallel()
	t.Skip("per-message recover と Extend ハートビートは engine 統合テスト TestEngine_Run/engineRunPanicIsolated・engineRunExtendCalledPeriodically でカバー")
}

func Test_run_startHeartbeat(t *testing.T) {
	t.Parallel()
	t.Skip("Extend ハートビートは engine 統合テスト TestEngine_Run/engineRunExtendCalledPeriodically・engineRunNoExtendWhenIntervalNonPositive でカバー")
}

func Test_run_handleResult(t *testing.T) {
	t.Parallel()
	t.Skip(
		"Handle 結果の Ack/Nack/Permanent/Fatal 振り分けは engine 統合テスト TestEngine_Run/engineRunAcksOnSuccess・engineRunRetryableNacked・engineRunPermanentGoesToFailureHandler・engineRunFatalStopsEngine でカバー",
	)
}

func Test_run_ack(t *testing.T) {
	t.Parallel()
	t.Skip("Ack とエラーログは engine 統合テスト TestEngine_Run/engineRunAcksOnSuccess・engineRunAckErrorLogged でカバー")
}

func Test_run_nack(t *testing.T) {
	t.Parallel()
	t.Skip("Nack と再配送 backoff は engine 統合テスト TestEngine_Run/engineRunRetryableNacked・engineRunNackErrorLogged でカバー")
}

func Test_run_drain(t *testing.T) {
	t.Parallel()
	t.Skip("in-flight の drain 待機は engine 統合テスト TestEngine_drain でカバー")
}
