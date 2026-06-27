package worker

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/boundary/worker/fake"
	"go-boilerplate/pkg/xerrors"
)

const (
	eventually = 2 * time.Second
	tick       = 5 * time.Millisecond
)

// --- テスト用ヘルパー ---

type handlerFunc func(ctx context.Context, m bw.Message) error

type testWorker struct {
	name    string
	cons    bw.Consumer
	handler bw.Handler
	failure bw.FailureHandler
}

func (h handlerFunc) Handle(ctx context.Context, m bw.Message) error { return h(ctx, m) }

func (w testWorker) Name() string { return w.name }

func (w testWorker) Consumer() bw.Consumer { return w.cons }

func (w testWorker) Handler() bw.Handler { return w.handler }

func (w testWorker) FailureHandler() bw.FailureHandler { return w.failure }

func baseSettings() Settings {
	return Settings{Concurrency: 4, MaxInFlight: 8, BatchSize: 4, DrainTimeout: 2 * time.Second}
}

// startEngine は、worker を別 goroutine で起動し、cancel と完了 channel を返します。
func startEngine(t *testing.T, set Settings, log logging.Logger, w bw.Worker) (context.CancelFunc, <-chan error) {
	t.Helper()
	eng, err := New([]bw.Worker{w}, set, observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), log)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx, w.Name()) }()
	return cancel, done
}

// --- New / Names / Run ---

func Test_New(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名の worker が重複登録された場合はエラー", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "dup", cons: fake.New(), handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			eng, err := New(
				[]bw.Worker{w, w},
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logging.NewTestLogger(t),
			)

			require.ErrorIs(t, err, ErrDuplicateWorker)
			assert.Nil(t, eng)
		})
	})
}

func Test_Engine_Run(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録の worker 名が指定された場合はエラー", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "known", cons: fake.New(), handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			eng, err := New(
				[]bw.Worker{w},
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logging.NewTestLogger(t),
			)
			require.NoError(t, err)

			err = eng.Run(context.Background(), "unknown")

			require.ErrorIs(t, err, ErrUnknownWorker)
		})
	})
}

// --- A. 配信correctness ---

func Test_Engine_A1_A2_AckNackDiscipline(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功時に Ack され Nack されない", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool { return len(f.AckedIDs()) == 1 }, eventually, tick)
			assert.Empty(t, f.NackedIDs())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Retryable 失敗時は Nack され Ack されない", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				return xerrors.Wrap(apperror.ErrRetryable, "downstream")
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool { return len(f.NackedIDs()) >= 1 }, eventually, tick)
			assert.Empty(t, f.AckedIDs())
			// M3: retryable は per-message backoff つきで再配送される（NackWithBackoff 経由）。
			assert.True(t, f.NackBackoffApplied("a"), "NackWithBackoff で再配送されること")
		})
	})
}

func Test_Engine_A3_ExtendHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("長時間処理中に Extend が周期的に呼ばれる", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			release := make(chan struct{})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				<-release
				return nil
			})}
			set := baseSettings()
			set.ExtendInterval = 5 * time.Millisecond

			cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
			defer func() { close(release); cancel(); <-done }()

			require.Eventually(t, func() bool { return f.ExtendCount("a") >= 2 }, eventually, tick)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Extend が失敗してもハートビートは継続し処理は完了する", func(t *testing.T) {
			t.Parallel()

			// H2: Extend 失敗は握り潰さず log+metric で可視化しつつ、ハートビート goroutine は
			// 生き続け、handler 完了で Ack される（lease 延長失敗は致命ではない）。
			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			f.SetExtendErr(xerrors.New("extend boom"))
			release := make(chan struct{})
			var once sync.Once
			closeRelease := func() { once.Do(func() { close(release) }) }
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				<-release
				return nil
			})}
			set := baseSettings()
			set.ExtendInterval = 5 * time.Millisecond

			cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
			defer func() { closeRelease(); cancel(); <-done }()

			require.Eventually(t, func() bool { return f.ExtendCount("a") >= 2 }, eventually, tick)
			closeRelease()
			require.Eventually(t, func() bool { return len(f.AckedIDs()) >= 1 }, eventually, tick)
		})
	})
}

func Test_Engine_A4_DuplicateDelivery(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一 ID の重複配送がそのまま Handler に届く", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"}, bw.Message{ID: "a"})
			var mu sync.Mutex
			count := 0
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				mu.Lock()
				count++
				mu.Unlock()
				return nil
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return count == 2
			}, eventually, tick)
		})
	})
}

func Test_Engine_A5_PermanentAndFatal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Permanent は FailureHandler へ退避してから Ack される", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			w := testWorker{name: "w", cons: f, failure: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				return xerrors.Wrap(apperror.ErrPermanent, "bad message")
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool {
				return len(f.Failed()) == 1 && len(f.AckedIDs()) == 1
			}, eventually, tick)
			assert.Empty(t, f.NackedIDs())
		})

		t.Run("Fatal は cancel 無しでも engine を停止させる", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			fatal := xerrors.Wrap(apperror.ErrFatal, "config broken")
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				return fatal
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer cancel()

			select {
			case err := <-done:
				require.ErrorIs(t, err, apperror.ErrFatal)
			case <-time.After(eventually):
				t.Fatal("Fatal で engine が停止しなかった")
			}
		})
	})
}

func Test_Engine_A6_PanicIsolation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 メッセージの panic が他メッセージ・engine を巻き込まない", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "good"}, bw.Message{ID: "bad"})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(_ context.Context, m bw.Message) error {
				if m.ID == "bad" {
					panic("boom")
				}
				return nil
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool { return slices.Contains(f.AckedIDs(), "good") }, eventually, tick)
			require.Eventually(t, func() bool { return slices.Contains(f.NackedIDs(), "bad") }, eventually, tick)
		})
	})
}

func Test_Engine_A7_PoisonWarn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ReceiveCount が閾値以上で warn ログが出る", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			set := baseSettings()
			set.ReceiveCountWarnThreshold = 1
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

			cancel, done := startEngine(t, set, logger, w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool {
				return observed.FilterMessage("receive count threshold reached").Len() >= 1
			}, eventually, tick)
		})
	})
}

// --- B. backpressure / 順序 ---

func Test_Engine_B1_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同時 Handle 数が Concurrency を超えない", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			for i := range 6 {
				f.Enqueue(bw.Message{ID: string(rune('a' + i))})
			}
			var cur, maxObserved atomic.Int32
			release := make(chan struct{})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				c := cur.Add(1)
				for {
					old := maxObserved.Load()
					if c <= old || maxObserved.CompareAndSwap(old, c) {
						break
					}
				}
				<-release
				cur.Add(-1)
				return nil
			})}
			set := baseSettings()
			set.Concurrency = 2
			set.MaxInFlight = 8
			set.BatchSize = 8

			cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
			defer func() { close(release); cancel(); <-done }()

			require.Eventually(t, func() bool { return cur.Load() == 2 }, eventually, tick)
			assert.LessOrEqual(t, maxObserved.Load(), int32(2))
		})
	})
}

func Test_Engine_B2_InflightLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("in-flight が MaxInFlight を超えない", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			for i := range 10 {
				f.Enqueue(bw.Message{ID: string(rune('a' + i))})
			}
			release := make(chan struct{})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				<-release
				return nil
			})}
			set := baseSettings()
			set.Concurrency = 3
			set.MaxInFlight = 3
			set.BatchSize = 5

			cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
			defer func() { close(release); cancel(); <-done }()

			require.Eventually(t, func() bool { return f.InflightLen() == 3 }, eventually, tick)
			assert.LessOrEqual(t, f.InflightLen(), 3)
		})
	})
}

func Test_Engine_B3_PartitionSerialization(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一 PartitionKey のメッセージは投入順に直列処理される", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(
				bw.Message{ID: "1", PartitionKey: "k"},
				bw.Message{ID: "2", PartitionKey: "k"},
				bw.Message{ID: "3", PartitionKey: "k"},
			)
			var mu sync.Mutex
			var order []string
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(_ context.Context, m bw.Message) error {
				time.Sleep(2 * time.Millisecond)
				mu.Lock()
				order = append(order, m.ID)
				mu.Unlock()
				return nil
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return len(order) == 3
			}, eventually, tick)

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"1", "2", "3"}, order)
		})
	})
}

func Test_Engine_B4_CircuitBreaker(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("連続失敗で Open し intake が止まり、回復後 Half-open で再開する", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			var failing atomic.Bool
			failing.Store(true)
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				if failing.Load() {
					return xerrors.Wrap(apperror.ErrRetryable, "downstream down")
				}
				return nil
			})}
			set := baseSettings()
			set.Concurrency = 1
			set.MaxInFlight = 1
			set.BatchSize = 1
			set.CircuitFailureThreshold = 2
			set.CircuitOpenBackoffInitial = 300 * time.Millisecond
			set.CircuitOpenBackoffMax = 300 * time.Millisecond

			cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
			defer func() { cancel(); <-done }()

			// B4-1: 連続 2 失敗で Open。cooldown 中は Receive されず Nack が増えない。
			require.Eventually(t, func() bool { return len(f.NackedIDs()) >= 2 }, eventually, tick)
			nacksAtOpen := len(f.NackedIDs())
			time.Sleep(80 * time.Millisecond) // cooldown(300ms) 未満
			assert.Len(t, f.NackedIDs(), nacksAtOpen)

			// B4-2: 回復させると cooldown 経過後の Half-open 試行で成功し Ack される。
			failing.Store(false)
			require.Eventually(t, func() bool { return len(f.AckedIDs()) >= 1 }, 3*time.Second, tick)
		})
	})
}

// --- C. lifecycle ---

func Test_Engine_C1_drain(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx cancel 後も in-flight は drain 期限内に完了して Ack される", func(t *testing.T) {
			t.Parallel()

			f := fake.New()
			f.Enqueue(bw.Message{ID: "a"})
			started := make(chan struct{}, 1)
			release := make(chan struct{})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				started <- struct{}{}
				<-release
				return nil
			})}

			cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)

			<-started // in-flight に入った
			cancel()  // Receive 停止
			close(release)

			select {
			case <-done:
			case <-time.After(eventually):
				t.Fatal("drain が完了しなかった")
			}
			assert.Equal(t, []string{"a"}, f.AckedIDs())
		})
	})
}
