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
	"go-boilerplate/pkg/xerrors"
)

func Test_newRun(t *testing.T) {
	t.Parallel()

	noop := handlerFunc(func(context.Context, bw.Message) error { return nil })

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker の seam と engine を保持する", func(t *testing.T) {
			t.Parallel()

			handled := false
			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, failure: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				handled = true
				return nil
			})}
			eng := newTestEngine(t, baseSettings(), w)

			r := newRun(eng, w)

			assert.Same(t, eng, r.e)
			assert.Equal(t, "w", r.name)
			assert.Same(t, f, r.consumer)
			assert.Same(t, f, r.failure)
			require.NoError(t, r.handler.Handle(context.Background(), bw.Message{ID: "a"}))
			assert.True(t, handled) // worker が返した Handler がそのまま保持されている
		})

		t.Run("Settings の上限値がセマフォと key キューの容量になる", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: noop}
			set := baseSettings()
			set.Concurrency = 2
			set.MaxInFlight = 5

			r := newRun(newTestEngine(t, set, w), w)

			assert.Equal(t, 5, cap(r.inflight))
			assert.Equal(t, 2, cap(r.conc))
			assert.Equal(t, 1, cap(r.slotFreed)) // 起床通知は取りこぼしても良いので容量 1
			assert.Equal(t, 5, r.keyed.buffer)
		})

		t.Run("サーキット設定が engine の Settings から渡る", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: noop}
			set := baseSettings()
			set.CircuitFailureThreshold = 3
			set.CircuitOpenBackoffInitial = 2 * time.Second
			set.CircuitOpenBackoffMax = 20 * time.Second
			eng := newTestEngine(t, set, w)

			r := newRun(eng, w)

			assert.Equal(t, 3, r.cb.threshold)
			assert.Equal(t, eng.set.circuitBackoff(), r.cb.backoff)
		})
	})
}

func Test_msSince(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("過去の時刻からの経過をミリ秒換算で返す", func(t *testing.T) {
			t.Parallel()

			got := msSince(time.Now().Add(-1500 * time.Millisecond))

			assert.GreaterOrEqual(t, got, 1500.0)
			assert.Less(t, got, 60000.0)
		})

		t.Run("未来の時刻を渡すと負のミリ秒を返す（0 へ丸めない）", func(t *testing.T) {
			t.Parallel()

			assert.Negative(t, msSince(time.Now().Add(time.Minute)))
		})
	})
}

func Test_run_nackBackoff(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		set := baseSettings()
		set.NackBackoffInitial = 10 * time.Millisecond
		set.NackBackoffMax = time.Second
		return newRun(newTestEngine(t, set, w), w)
	}

	// full jitter で散らされるため、値そのものではなく指数 backoff の算出値を上限として判定する。
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回配送（ReceiveCount=1）は初期 backoff を上限とする", func(t *testing.T) {
			t.Parallel()

			got := newTestRun(t).nackBackoff(1)

			assert.GreaterOrEqual(t, got, time.Duration(0))
			assert.LessOrEqual(t, got, 10*time.Millisecond)
		})

		t.Run("再配送回数に応じて上限が指数的に伸びる", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)

			// ReceiveCount=3 は attempt=2 なので上限は 10ms * 2^2 = 40ms。full jitter は [0, 上限] の
			// 一様乱数なので、上限が初期値 10ms のままなら 200 回引いて 10ms を超えない確率は事実上 0。
			var maxSeen time.Duration
			for range 200 {
				got := r.nackBackoff(3)
				require.LessOrEqual(t, got, 40*time.Millisecond)
				maxSeen = max(maxSeen, got)
			}

			assert.Greater(t, maxSeen, 10*time.Millisecond)
		})

		t.Run("ReceiveCount が 0 以下でも attempt は 0 に丸められる", func(t *testing.T) {
			t.Parallel()

			// 0 起算へ変換すると負になる入力でも backoff.Duration へ負の attempt を渡さない。
			assert.LessOrEqual(t, newTestRun(t).nackBackoff(0), 10*time.Millisecond)
			assert.LessOrEqual(t, newTestRun(t).nackBackoff(-5), 10*time.Millisecond)
		})
	})
}

func Test_run_fatalErr(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		return newRun(newTestEngine(t, baseSettings(), w), w)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Fatal が未記録の場合は nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, newTestRun(t).fatalErr())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("記録済みの Fatal をそのまま返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRun(t)
			boom := xerrors.New("boom")
			r.fatalMu.Lock()
			r.fatal = boom
			r.fatalMu.Unlock()

			require.ErrorIs(t, r.fatalErr(), boom)
		})
	})
}

func Test_run_logErr(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した logger 名と error フィールド付きで Error ログに残す", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			eng, err := New(
				[]bw.Worker{w},
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logger,
			)
			require.NoError(t, err)
			r := newRun(eng, w)

			r.logErr(context.Background(), "worker.probe", "probe failed", xerrors.New("boom"))

			entries := observed.FilterMessage("probe failed").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "error", entries[0].Level.String())
			assert.Equal(t, "worker.probe", entries[0].LoggerName)
			assert.Contains(t, entries[0].ContextMap()[logging.ErrorKey], "boom")
		})
	})
}
