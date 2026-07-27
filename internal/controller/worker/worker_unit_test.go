package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
	"go-boilerplate/pkg/backoff"
	"go-boilerplate/pkg/xerrors"
)

// newTestEngine は、noop な tracer/metrics/logger を注入した Engine を生成します。
func newTestEngine(t *testing.T, set Settings, w bw.Worker) *Engine {
	t.Helper()
	eng, err := New(
		[]bw.Worker{w},
		set,
		observability.NewNoopTracerFactory(t),
		observability.NewNoopWorkerMetrics(t),
		logging.NewTestLogger(t),
	)
	require.NoError(t, err)
	return eng
}

func TestEngine_Names(t *testing.T) {
	t.Parallel()

	noop := handlerFunc(func(context.Context, bw.Message) error { return nil })

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録された worker 名がソート順で返る", func(t *testing.T) {
			t.Parallel()

			w1 := testWorker{name: "c", cons: testkit.NewFake(), handler: noop}
			w2 := testWorker{name: "a", cons: testkit.NewFake(), handler: noop}
			w3 := testWorker{name: "b", cons: testkit.NewFake(), handler: noop}
			eng, err := New(
				[]bw.Worker{w1, w2, w3},
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logging.NewTestLogger(t),
			)
			require.NoError(t, err)

			assert.Equal(t, []string{"a", "b", "c"}, eng.Names())
		})

		t.Run("worker が 0 件の場合は空スライスを返す", func(t *testing.T) {
			t.Parallel()

			eng, err := New(
				nil,
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logging.NewTestLogger(t),
			)
			require.NoError(t, err)

			assert.Empty(t, eng.Names())
		})
	})
}

func TestEngine_Healthy(t *testing.T) {
	t.Parallel()

	noop := handlerFunc(func(context.Context, bw.Message) error { return nil })

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Run 実行中で進捗が新鮮な場合は true", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "w", cons: testkit.NewFake(), handler: noop}
			eng := newTestEngine(t, baseSettings(), w)
			eng.active.Store(true)
			eng.markProgress()

			assert.True(t, eng.Healthy())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Run 未実行の場合は false", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "w", cons: testkit.NewFake(), handler: noop}
			eng := newTestEngine(t, baseSettings(), w)

			assert.False(t, eng.Healthy())
		})

		t.Run("Run 実行中でも進捗が古い場合は false", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "w", cons: testkit.NewFake(), handler: noop}
			set := baseSettings()
			set.ProgressStaleAfter = 10 * time.Millisecond
			eng := newTestEngine(t, set, w)
			eng.active.Store(true)
			eng.progress.Store(time.Now().Add(-time.Second).UnixNano())

			assert.False(t, eng.Healthy())
		})
	})
}

func TestSettings_normalize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値の場合は全項目が既定値で補完される", func(t *testing.T) {
			t.Parallel()

			s := Settings{}
			s.normalize()

			assert.Equal(t, 1, s.Concurrency)
			assert.Equal(t, 1, s.MaxInFlight)
			assert.Equal(t, 1, s.BatchSize)
			assert.Equal(t, defaultDrainTimeout, s.DrainTimeout)
			assert.Equal(t, 1, s.CircuitHalfOpenProbe)
			assert.Equal(t, defaultProgressStaleAfter, s.ProgressStaleAfter)
			assert.Equal(t, defaultNackBackoffInitial, s.NackBackoffInitial)
			assert.Equal(t, defaultNackBackoffMax, s.NackBackoffMax)
			assert.Equal(t, defaultCircuitOpenBackoffInitial, s.CircuitOpenBackoffInitial)
			assert.Equal(t, defaultCircuitOpenBackoffMax, s.CircuitOpenBackoffMax)
		})

		t.Run("MaxInFlight が Concurrency 未満の場合は Concurrency まで引き上げる", func(t *testing.T) {
			t.Parallel()

			s := Settings{Concurrency: 5, MaxInFlight: 2, BatchSize: 1}
			s.normalize()

			assert.Equal(t, 5, s.MaxInFlight)
		})

		t.Run("BatchSize が MaxInFlight を超える場合は MaxInFlight まで切り詰める", func(t *testing.T) {
			t.Parallel()

			s := Settings{Concurrency: 2, MaxInFlight: 4, BatchSize: 10}
			s.normalize()

			assert.Equal(t, 4, s.BatchSize)
		})

		t.Run("BatchSize が 1 未満の場合は 1 に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{Concurrency: 2, MaxInFlight: 4, BatchSize: 0}
			s.normalize()

			assert.Equal(t, 1, s.BatchSize)
		})

		t.Run("CircuitHalfOpenProbe が 1 未満の場合は 1 に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{CircuitHalfOpenProbe: 0}
			s.normalize()

			assert.Equal(t, 1, s.CircuitHalfOpenProbe)
		})

		t.Run("ProgressStaleAfter が 0 以下の場合は既定値に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{ProgressStaleAfter: -1}
			s.normalize()

			assert.Equal(t, defaultProgressStaleAfter, s.ProgressStaleAfter)
		})

		t.Run("NackBackoffInitial が 0 以下の場合は既定値に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{NackBackoffInitial: -1}
			s.normalize()

			assert.Equal(t, defaultNackBackoffInitial, s.NackBackoffInitial)
		})

		t.Run("NackBackoffMax が 0 以下の場合は既定値に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{NackBackoffMax: -1}
			s.normalize()

			assert.Equal(t, defaultNackBackoffMax, s.NackBackoffMax)
		})

		t.Run("CircuitOpenBackoffInitial が 0 以下の場合は既定値に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{CircuitOpenBackoffInitial: -1}
			s.normalize()

			assert.Equal(t, defaultCircuitOpenBackoffInitial, s.CircuitOpenBackoffInitial)
		})

		t.Run("CircuitOpenBackoffMax が 0 以下の場合は既定値に補完する", func(t *testing.T) {
			t.Parallel()

			s := Settings{CircuitOpenBackoffMax: -1}
			s.normalize()

			assert.Equal(t, defaultCircuitOpenBackoffMax, s.CircuitOpenBackoffMax)
		})

		t.Run("有効な値が指定されている場合はそのまま保持する", func(t *testing.T) {
			t.Parallel()

			s := Settings{
				Concurrency:          3,
				MaxInFlight:          10,
				BatchSize:            5,
				DrainTimeout:         10 * time.Second,
				CircuitHalfOpenProbe: 3,
				ProgressStaleAfter:   5 * time.Second,
				NackBackoffInitial:   2 * time.Second,
				NackBackoffMax:       10 * time.Second,
			}
			s.normalize()

			assert.Equal(t, 3, s.Concurrency)
			assert.Equal(t, 10, s.MaxInFlight)
			assert.Equal(t, 5, s.BatchSize)
			assert.Equal(t, 10*time.Second, s.DrainTimeout)
			assert.Equal(t, 3, s.CircuitHalfOpenProbe)
			assert.Equal(t, 5*time.Second, s.ProgressStaleAfter)
			assert.Equal(t, 2*time.Second, s.NackBackoffInitial)
			assert.Equal(t, 10*time.Second, s.NackBackoffMax)
		})
	})
}

func TestNewState(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成直後の Snapshot はゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			s := NewState()
			require.NotNil(t, s)

			name, args, done := s.Snapshot()
			assert.Empty(t, name)
			assert.Nil(t, args)
			assert.Nil(t, done)
		})
	})
}

func Test_state_Set(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定した worker 名・引数・done が Snapshot で取得できる", func(t *testing.T) {
			t.Parallel()

			s := NewState()
			done := make(chan error, 1)
			s.Set("w", []string{"--flag"}, done)

			name, args, gotDone := s.Snapshot()
			assert.Equal(t, "w", name)
			assert.Equal(t, []string{"--flag"}, args)
			assert.Equal(t, done, gotDone)
		})
	})
}

func Test_state_Snapshot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set 前はゼロ値、Set 後は設定値を返す", func(t *testing.T) {
			t.Parallel()

			s := NewState()
			name, args, done := s.Snapshot()
			assert.Empty(t, name)
			assert.Nil(t, args)
			assert.Nil(t, done)

			ch := make(chan error, 1)
			s.Set("worker", []string{"a", "b"}, ch)

			name, args, done = s.Snapshot()
			assert.Equal(t, "worker", name)
			assert.Equal(t, []string{"a", "b"}, args)
			assert.Equal(t, ch, done)
		})
	})
}

func Test_classify(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrFatal をラップした error は catFatal に分類する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, catFatal, classify(xerrors.Wrap(apperror.ErrFatal, "config broken")))
		})

		t.Run("ErrPermanent をラップした error は catPermanent に分類する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, catPermanent, classify(xerrors.Wrap(apperror.ErrPermanent, "bad message")))
		})

		t.Run("ErrRetryable をラップした error は catRetryable に分類する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, catRetryable, classify(xerrors.Wrap(apperror.ErrRetryable, "downstream")))
		})

		t.Run("いずれのセンチネルにも該当しない error は catRetryable に分類する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, catRetryable, classify(xerrors.New("unclassified")))
		})
	})
}

func Test_circuit_onFailure(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Closed で連続失敗が閾値に達すると Open へ遷移する", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(2, bo)
			c.onFailure()
			assert.Equal(t, phaseClosed, c.phaseNow())

			c.onFailure()
			assert.Equal(t, phaseOpen, c.phaseNow())
		})

		t.Run("Half-open での失敗は即 Open へ戻す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()
			c.toHalfOpen()
			require.Equal(t, phaseHalfOpen, c.phaseNow())

			c.onFailure()
			assert.Equal(t, phaseOpen, c.phaseNow())
		})

		t.Run("Open 中の失敗では状態と cooldown が変化しない", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()
			require.Equal(t, phaseOpen, c.phaseNow())
			cdBefore := c.cooldown()

			c.onFailure()
			assert.Equal(t, phaseOpen, c.phaseNow())
			assert.Equal(t, cdBefore, c.cooldown())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("threshold が 0 以下の場合は連続失敗しても Open しない", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(0, bo)
			for range 5 {
				c.onFailure()
			}

			assert.Equal(t, phaseClosed, c.phaseNow())
		})
	})
}

func Test_run_routePermanent(t *testing.T) {
	t.Parallel()

	cause := xerrors.Wrap(apperror.ErrPermanent, "bad message")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("FailureHandler 未設定の場合は直接 Ack する", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, baseSettings(), w), w)

			err := r.routePermanent(context.Background(), bw.Message{ID: "a"}, cause)

			require.NoError(t, err)
			assert.Equal(t, []string{"a"}, f.AckedIDs())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("FailureHandler が失敗した場合は Ack せずその error を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fh := mock_worker.NewMockFailureHandler(ctrl)
			boom := xerrors.New("failure boom")
			fh.EXPECT().Fail(gomock.Any(), gomock.Any(), gomock.Any()).Return(boom)

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, failure: fh, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, baseSettings(), w), w)

			err := r.routePermanent(context.Background(), bw.Message{ID: "a"}, cause)

			require.ErrorIs(t, err, boom)
			assert.Empty(t, f.AckedIDs())
		})
	})
}

func Test_run_onPollError(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Receive 失敗をエラーログに残しサーキットへ失敗を計上する", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			set := baseSettings()
			set.CircuitFailureThreshold = 1 // 1 度の poll 失敗で Open へ遷移させる
			eng, err := New([]bw.Worker{w}, set, observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), logger)
			require.NoError(t, err)
			r := newRun(eng, w)

			r.onPollError(context.Background(), xerrors.New("broker unreachable"))

			assert.Equal(t, 1, observed.FilterMessage("receive error").Len())
			assert.Equal(t, phaseOpen, r.cb.phaseNow())
		})
	})
}

func Test_run_dispatchAll(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信メッセージが 0 件の場合は何も dispatch せず返る", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, baseSettings(), w), w)

			r.dispatchAll(context.Background(), nil)

			assert.Empty(t, r.inflight) // in-flight トークンが 1 つも計上されないこと
		})
	})
}

func Test_run_waitForSlot(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		return newRun(newTestEngine(t, baseSettings(), w), w)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("in-flight に空きがある場合は即座に空き数を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)

			free, ok := r.waitForSlot(context.Background())

			assert.True(t, ok)
			assert.Equal(t, cap(r.inflight), free)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("in-flight が満杯かつ ctx がキャンセル済みの場合は中断する", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			for range cap(r.inflight) {
				r.inflight <- struct{}{}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			free, ok := r.waitForSlot(ctx)

			assert.False(t, ok)
			assert.Zero(t, free)
		})
	})
}

func Test_msgFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker 名・message id・receive count を正しいキーと値で返す（trace_id は Logger が注入する）", func(t *testing.T) {
			t.Parallel()

			fields := msgFields("w", bw.Message{ID: "a", ReceiveCount: 2})

			require.Len(t, fields, 3)
			assert.Equal(t, logging.String(logging.WorkerNameKey, "w"), fields[0])
			assert.Equal(t, logging.String(logging.MessageIDKey, "a"), fields[1])
			assert.Equal(t, logging.Int(logging.ReceiveCountKey, 2), fields[2])
		})
	})
}

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

func Test_circuit_tryBeginProbe(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Half-open では最初の 1 回だけ true を返し、以降は probing 中として false を返す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()  // Closed -> Open
			c.toHalfOpen() // Open -> Half-open
			require.Equal(t, phaseHalfOpen, c.phaseNow())

			assert.True(t, c.tryBeginProbe())  // 1 バッチ目のみ投入許可
			assert.False(t, c.tryBeginProbe()) // 以降は probing 中で拒否
			assert.False(t, c.tryBeginProbe())
		})

		t.Run("次の Half-open エピソード（toHalfOpen）で probing がリセットされ再び投入できる", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()
			c.toHalfOpen()
			require.True(t, c.tryBeginProbe())
			require.False(t, c.tryBeginProbe())

			c.onFailure()  // Half-open 失敗 -> Open（trip）
			c.toHalfOpen() // 次のエピソード

			assert.True(t, c.tryBeginProbe()) // リセットされ再投入可
		})

		t.Run("Half-open 成功で Closed 復帰後は probing がリセットされる", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()
			c.toHalfOpen()
			require.True(t, c.tryBeginProbe())

			c.onSuccess() // Half-open -> Closed
			require.Equal(t, phaseClosed, c.phaseNow())

			c.onFailure()  // 再び Open
			c.toHalfOpen() // 次の Half-open
			assert.True(t, c.tryBeginProbe())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Half-open 以外（Closed / Open）では false を返す", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(2, bo)
			require.Equal(t, phaseClosed, c.phaseNow())
			assert.False(t, c.tryBeginProbe()) // Closed

			c.onFailure()
			c.onFailure() // threshold=2 -> Open
			require.Equal(t, phaseOpen, c.phaseNow())
			assert.False(t, c.tryBeginProbe()) // Open
		})
	})
}

func Test_circuit_abortProbe(t *testing.T) {
	t.Parallel()

	bo := backoff.Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Half-open の probing 中に呼ぶと probing を解除し再 probe できる", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			c.onFailure()                       // Closed -> Open
			c.toHalfOpen()                      // Open -> Half-open
			require.True(t, c.tryBeginProbe())  // probing=true
			require.False(t, c.tryBeginProbe()) // 既に probing 中

			c.abortProbe()

			assert.Equal(t, phaseHalfOpen, c.phaseNow()) // phase は Half-open のまま
			assert.True(t, c.tryBeginProbe())            // probing 解除済みなので再 probe 可
		})

		t.Run("Half-open 以外（Closed / Open）では no-op", func(t *testing.T) {
			t.Parallel()

			c := newCircuit(1, bo)
			require.Equal(t, phaseClosed, c.phaseNow())
			c.abortProbe()
			assert.Equal(t, phaseClosed, c.phaseNow())

			c.onFailure() // Open
			require.Equal(t, phaseOpen, c.phaseNow())
			c.abortProbe()
			assert.Equal(t, phaseOpen, c.phaseNow())
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

func Test_run_acquire(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		set := baseSettings()
		set.MaxInFlight = 10
		set.BatchSize = 10
		set.CircuitHalfOpenProbe = 2
		set.CircuitFailureThreshold = 1
		return newRun(newTestEngine(t, set, w), w)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// MaxInFlight(10) > CircuitHalfOpenProbe(2) でも、Half-open では probe バッチを 1 度だけ投入し、
		// 結果が確定するまで新規 Receive を止める（総 probe ≤ CircuitHalfOpenProbe の回帰）。
		t.Run("Half-open では probe を 1 バッチだけ投入し、結果確定まで再投入せず、確定後は通常 Receive を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			r.cb.onFailure()  // Open へ
			r.cb.toHalfOpen() // Half-open へ
			require.Equal(t, phaseHalfOpen, r.cb.phaseNow())

			// 1 回目: MaxInFlight=10 でも probe=2 に制限される。
			n, ok := r.acquire(context.Background())
			require.True(t, ok)
			require.Equal(t, 2, n)

			// probe が in-flight にある状態を再現する。
			r.inflight <- struct{}{}
			r.inflight <- struct{}{}

			// 2 回目: probing 中は再投入せずブロックするため goroutine で回す。
			type acqResult struct {
				n  int
				ok bool
			}
			ch := make(chan acqResult, 1)
			go func() {
				gotN, gotOK := r.acquire(context.Background())
				ch <- acqResult{gotN, gotOK}
			}()

			// probe を再投入していない（＝ブロックしている）ことを確認する。
			select {
			case <-ch:
				t.Fatal("probing 中に acquire が新規 Receive を返した（probe が再投入された）")
			case <-time.After(50 * time.Millisecond):
			}

			// probe 結果の到来を再現: 成功で Closed へ遷移し in-flight を 1 つ解放する。
			r.cb.onSuccess()
			<-r.inflight
			select {
			case r.slotFreed <- struct{}{}:
			default:
			}

			// 起床して Closed の通常 Receive（probe バッチではない）を返す。
			select {
			case got := <-ch:
				assert.True(t, got.ok)
				// Closed の通常 Receive は min(BatchSize=10, free) を返す。probe で 2 つ充填し 1 つ解放したため
				// free = cap(10) - 1 = 9 となり、min(10, 9) = 9 に決まる。
				assert.Equal(t, 9, got.n)
			case <-time.After(time.Second):
				t.Fatal("結果確定後に acquire が復帰しなかった")
			}
			assert.Equal(t, phaseClosed, r.cb.phaseNow())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx が完了している場合は Receive せず false を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			n, ok := r.acquire(ctx)

			assert.False(t, ok)
			assert.Zero(t, n)
		})

		t.Run("probing 待機中に ctx がキャンセルされると Receive せず false を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			r.cb.onFailure()  // Open へ
			r.cb.toHalfOpen() // Half-open へ
			require.Equal(t, phaseHalfOpen, r.cb.phaseNow())

			// probe バッチを投入して probing 中にする（in-flight を充填して slotFreed が鳴らない状態）。
			n, ok := r.acquire(context.Background())
			require.True(t, ok)
			require.Equal(t, 2, n)
			r.inflight <- struct{}{}
			r.inflight <- struct{}{}

			ctx, cancel := context.WithCancel(context.Background())
			type acqResult struct {
				n  int
				ok bool
			}
			ch := make(chan acqResult, 1)
			go func() {
				gotN, gotOK := r.acquire(ctx)
				ch <- acqResult{gotN, gotOK}
			}()

			// probing 中は結果待ちでブロックする。
			select {
			case <-ch:
				t.Fatal("probing 待機中に acquire が復帰した")
			case <-time.After(50 * time.Millisecond):
			}

			cancel() // slotFreed には触れず ctx キャンセルで抜ける

			select {
			case got := <-ch:
				assert.False(t, got.ok)
				assert.Zero(t, got.n)
			case <-time.After(time.Second):
				t.Fatal("ctx キャンセル後も acquire が復帰しなかった")
			}
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

	// poll loop の基本振り分け（ctx cancel / Ack / Nack / Fatal 等）は engine 統合テスト TestEngine_Run で
	// カバーする。ここでは Fake が空スライスを返せず到達不能な「Half-open の probe Receive が 0 件 →
	// abortProbe → 次周で再 probe」の配線を、生成 mock で決定的に固定する（abortProbe を外すと probing が
	// 解除されず tryBeginProbe が false のまま新規 Receive が止まり、成功メッセージが永久に届かない半開デッドロック）。
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Half-openのprobeが0件でも再probeして成功メッセージをAckする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mc := mock_worker.NewMockConsumer(ctrl)

			var mu sync.Mutex
			step := 0
			mc.EXPECT().Receive(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
				func(ctx context.Context, _ int) ([]bw.Message, error) {
					mu.Lock()
					step++
					s := step
					mu.Unlock()
					switch s {
					case 1:
						return nil, xerrors.New("broker unreachable") // poll 失敗で circuit を Open へ trip
					case 2:
						return []bw.Message{}, nil // Half-open の probe が空振り → abortProbe 経路
					case 3:
						return []bw.Message{{ID: "a"}}, nil // 再 probe で成功メッセージが届く
					default:
						<-ctx.Done() // 以降は intake を止め、ctx キャンセルで loop を終了させる
						return nil, ctx.Err()
					}
				})

			acked := make(chan string, 1)
			mc.EXPECT().Ack(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
				func(_ context.Context, m bw.Message) error {
					select {
					case acked <- m.ID:
					default:
					}
					return nil
				})

			set := baseSettings()
			set.Concurrency = 1
			set.MaxInFlight = 1
			set.BatchSize = 1
			set.CircuitHalfOpenProbe = 1
			set.CircuitFailureThreshold = 1 // 1 度の poll 失敗で Open へ
			set.CircuitOpenBackoffInitial = 10 * time.Millisecond
			set.CircuitOpenBackoffMax = 10 * time.Millisecond
			w := testWorker{name: "w", cons: mc, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, set, w), w)

			ctx, cancel := context.WithCancel(context.Background())
			loopDone := make(chan error, 1)
			go func() { loopDone <- r.loop(ctx) }()

			select {
			case id := <-acked:
				assert.Equal(t, "a", id)
			case <-time.After(eventually):
				t.Fatal("Half-open の再 probe が行われず成功メッセージが Ack されなかった（半開デッドロック）")
			}

			cancel()
			select {
			case <-loopDone:
			case <-time.After(eventually):
				t.Fatal("ctx キャンセルで loop が終了しなかった")
			}
		})
	})
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

	// Extend の周期発火（ticker.C 経路）と done 停止経路は engine 統合テスト
	// TestEngine_Run/engineRunExtendCalledPeriodically・engineRunExtendErrorHeartbeatContinues でカバーする。
	// ここでは engine 経由では done と競合して非決定になる ctx.Done() 停止経路を、ticker を実質無効化して
	// 決定的に検証する（停止クロージャが goroutine 終了を待つ契約も併せて確認する）。
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctxキャンセルでハートビートgoroutineが停止する", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			set := baseSettings()
			set.ExtendInterval = time.Hour // ticker.C を発火させず、停止トリガを ctx.Done() のみに限定する
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, set, w), w)

			ctx, cancel := context.WithCancel(context.Background())
			stop := r.startHeartbeat(ctx, bw.Message{ID: "a"})
			cancel() // ctx.Done() を唯一の ready case にして goroutine を停止させる
			stop()   // goroutine の完全終了（stopped の close）を待つ

			assert.Equal(t, 0, f.ExtendCount("a")) // ticker 未発火のため Extend は呼ばれない
		})

		t.Run("ExtendIntervalが0以下の場合はハートビートを起動しない", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			set := baseSettings()
			set.ExtendInterval = 0
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, set, w), w)

			stop := r.startHeartbeat(context.Background(), bw.Message{ID: "a"})
			stop() // no-op クロージャ（goroutine を起動していない）

			assert.Equal(t, 0, f.ExtendCount("a"))
		})
	})
}

func Test_run_handleResult(t *testing.T) {
	t.Parallel()

	// Ack/Nack/Fatal の基本振り分けは engine 統合テスト（engineRunAcksOnSuccess 等）でカバーする。
	// ここでは Fake が Fail エラーを注入できず circuit 状態も engine から観測しにくい
	// 「Permanent の dead-letter 退避成否による circuit / Ack 分岐」に絞って検証する。
	permErr := xerrors.Wrap(apperror.ErrPermanent, "bad message")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Permanentは退避成功でAckしcircuitをOpenしない", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			set := baseSettings()
			set.CircuitFailureThreshold = 1
			w := testWorker{name: "w", cons: f, failure: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, set, w), w)

			r.handleResult(context.Background(), bw.Message{ID: "a"}, permErr)

			assert.Equal(t, []string{"a"}, f.AckedIDs())
			assert.Empty(t, f.NackedIDs())
			assert.Equal(t, phaseClosed, r.cb.phaseNow())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Permanentは退避失敗でAckせずcircuitへ失敗計上する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fh := mock_worker.NewMockFailureHandler(ctrl)
			fh.EXPECT().Fail(gomock.Any(), gomock.Any(), gomock.Any()).Return(xerrors.New("dead-letter store down"))

			f := testkit.NewFake()
			set := baseSettings()
			set.CircuitFailureThreshold = 1 // 1 度の退避失敗で Open へ遷移させる
			w := testWorker{name: "w", cons: f, failure: fh, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, set, w), w)

			r.handleResult(context.Background(), bw.Message{ID: "a"}, permErr)

			assert.Empty(t, f.AckedIDs())  // 退避できていないので Ack しない
			assert.Empty(t, f.NackedIDs()) // Nack もしない（暗黙の再配送へ委ねる）
			assert.Equal(t, phaseOpen, r.cb.phaseNow())
		})
	})
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
