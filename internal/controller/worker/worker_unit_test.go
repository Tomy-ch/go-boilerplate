package worker

import (
	"context"
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

func Test_Engine_Names(t *testing.T) {
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

func Test_Engine_Healthy(t *testing.T) {
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

func Test_Settings_normalize(t *testing.T) {
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

func Test_NewState(t *testing.T) {
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

			r.routePermanent(context.Background(), bw.Message{ID: "a"}, cause)

			assert.Equal(t, []string{"a"}, f.AckedIDs())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("FailureHandler が失敗した場合は Ack しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fh := mock_worker.NewMockFailureHandler(ctrl)
			fh.EXPECT().Fail(gomock.Any(), gomock.Any(), gomock.Any()).Return(xerrors.New("failure boom"))

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, failure: fh, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			r := newRun(newTestEngine(t, baseSettings(), w), w)

			r.routePermanent(context.Background(), bw.Message{ID: "a"}, cause)

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
