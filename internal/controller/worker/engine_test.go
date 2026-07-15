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
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
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

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数の worker が正常登録され Engine を返す", func(t *testing.T) {
			t.Parallel()

			noop := handlerFunc(func(context.Context, bw.Message) error { return nil })
			w1 := testWorker{name: "b", cons: testkit.NewFake(), handler: noop}
			w2 := testWorker{name: "a", cons: testkit.NewFake(), handler: noop}
			eng, err := New(
				[]bw.Worker{w1, w2},
				baseSettings(),
				observability.NewNoopTracerFactory(t),
				observability.NewNoopWorkerMetrics(t),
				logging.NewTestLogger(t),
			)

			require.NoError(t, err)
			require.NotNil(t, eng)
			assert.Equal(t, []string{"a", "b"}, eng.Names())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名の worker が重複登録された場合はエラー", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "dup", cons: testkit.NewFake(), handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
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

// (*Engine).Run は 1 subject = 1 TestXxx の規約に従い単一のテスト関数へ集約している。
// Run の多数の挙動シナリオは正常系/異常系のサブテストとして束ね、各サブテスト本体は
// トップレベルのヘルパー関数へ抽出することで本関数の認知的複雑度を閾値未満に保つ。
func TestEngine_Run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx が cancel されると nil を返して終了する", engineRunCtxCancelReturnsNil)
		t.Run("成功時に Ack され Nack されない", engineRunAcksOnSuccess)
		t.Run("長時間処理中に Extend が周期的に呼ばれる", engineRunExtendCalledPeriodically)
		t.Run("ExtendInterval が 0 以下の場合は Extend が呼ばれない", engineRunNoExtendWhenIntervalNonPositive)
		t.Run("同一 ID の重複配送がそのまま Handler に届く", engineRunDuplicateDeliveryReachesHandler)
		t.Run("Permanent は FailureHandler へ退避してから Ack される", engineRunPermanentGoesToFailureHandler)
		t.Run("Fatal は cancel 無しでも engine を停止させる", engineRunFatalStopsEngine)
		t.Run("1 メッセージの panic が他メッセージ・engine を巻き込まない", engineRunPanicIsolated)
		t.Run("ReceiveCount が閾値以上で warn ログが出る", engineRunReceiveCountWarnLog)
		t.Run("同時 Handle 数が Concurrency を超えない", engineRunConcurrencyBounded)
		t.Run("in-flight が MaxInFlight を超えない", engineRunMaxInFlightBounded)
		t.Run("同一 PartitionKey のメッセージは投入順に直列処理される", engineRunPartitionKeySerialized)
		t.Run("連続失敗で Open し intake が止まり、回復後 Half-open で再開する", engineRunCircuitOpenAndRecover)
		t.Run("失敗ログに message id と receive count が構造化フィールドで残る", engineRunFailureLogStructuredFields)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録の worker 名が指定された場合はエラー", engineRunUnknownWorkerError)
		t.Run("Receive が一時的に失敗しても poll を継続し後続メッセージを処理する", engineRunReceiveFailureContinuesPoll)
		t.Run("Retryable 失敗時は Nack され Ack されない", engineRunRetryableNacked)
		t.Run("Ack が失敗した場合はエラーログを出力する", engineRunAckErrorLogged)
		t.Run("Nack が失敗した場合はエラーログを出力する", engineRunNackErrorLogged)
		t.Run("Extend が失敗してもハートビートは継続し処理は完了する", engineRunExtendErrorHeartbeatContinues)
		t.Run("停止中の Extend 失敗はログ・metric を出さず握り潰す", engineRunExtendErrorAfterStopSuppressed)
		t.Run("Open の cooldown 待機中に ctx がキャンセルされると acquire が中断する", engineRunAcquireInterruptedByCancel)
	})
}

// --- TestEngine_Run 正常系ヘルパー ---

func engineRunCtxCancelReturnsNil(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

	cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(eventually):
		t.Fatal("ctx cancel で engine が終了しなかった")
	}
}

func engineRunAcksOnSuccess(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
	f.Enqueue(bw.Message{ID: "a"})
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

	cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
	defer func() { cancel(); <-done }()

	require.Eventually(t, func() bool { return len(f.AckedIDs()) == 1 }, eventually, tick)
	assert.Empty(t, f.NackedIDs())
}

func engineRunExtendCalledPeriodically(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunNoExtendWhenIntervalNonPositive(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
	set := baseSettings()
	set.ExtendInterval = 0

	r := newRun(newTestEngine(t, set, w), w)
	stop := r.startHeartbeat(context.Background(), bw.Message{ID: "a"})
	stop()

	assert.Equal(t, 0, f.ExtendCount("a"))
}

func engineRunDuplicateDeliveryReachesHandler(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunPermanentGoesToFailureHandler(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
	require.ErrorIs(t, f.Failed()[0].Cause, apperror.ErrPermanent)
}

func engineRunFatalStopsEngine(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunPanicIsolated(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
	// panic は Retryable に変換され、per-message backoff つきで再配送される。
	assert.True(t, f.NackBackoffApplied("bad"), "NackWithBackoff で再配送されること")
}

func engineRunReceiveCountWarnLog(t *testing.T) {
	t.Parallel()

	logger, observed := logging.NewObservedTestLogger(t)
	f := testkit.NewFake()
	f.Enqueue(bw.Message{ID: "a"})
	set := baseSettings()
	set.ReceiveCountWarnThreshold = 1
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

	cancel, done := startEngine(t, set, logger, w)
	defer func() { cancel(); <-done }()

	require.Eventually(t, func() bool {
		return observed.FilterMessage("receive count threshold reached").Len() >= 1
	}, eventually, tick)
}

func engineRunConcurrencyBounded(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunMaxInFlightBounded(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunPartitionKeySerialized(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
}

func engineRunCircuitOpenAndRecover(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
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
	// cooldown を 1s 級へ引き上げ、-race のスローダウン下でも観測窓に Half-open probe が割り込まないようにする。
	set.CircuitOpenBackoffInitial = time.Second
	set.CircuitOpenBackoffMax = time.Second

	cancel, done := startEngine(t, set, logging.NewTestLogger(t), w)
	defer func() { cancel(); <-done }()

	// B4-1: 連続 2 失敗で Open へ遷移する。MaxInFlight=1 のため poll loop は失敗確定
	// （onFailure で trip）まで次の Receive に進めず、Open 遷移後は cooldown 待機に入る。
	require.Eventually(t, func() bool { return len(f.NackedIDs()) >= 2 }, eventually, tick)
	// Open 中は intake が止まり Receive されないため、cooldown（1s）より十分短い窓で
	// Nack も Ack も増えないことを確認する（実時間 sleep には依存しない）。
	require.Never(t, func() bool {
		return len(f.NackedIDs()) > 2 || len(f.AckedIDs()) > 0
	}, 200*time.Millisecond, tick)
	// 再配送された "a" が Receive されず queue に滞留していることが intake 停止の証跡。
	assert.Equal(t, 1, f.QueueLen())

	// B4-2: 回復させると cooldown 経過後の Half-open 試行で成功し Ack される。
	failing.Store(false)
	require.Eventually(t, func() bool { return len(f.AckedIDs()) >= 1 }, 5*time.Second, tick)
}

func engineRunFailureLogStructuredFields(t *testing.T) {
	t.Parallel()

	logger, observed := logging.NewObservedTestLogger(t)
	f := testkit.NewFake()
	f.Enqueue(bw.Message{ID: "a"})
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
		return xerrors.Wrap(apperror.ErrRetryable, "downstream")
	})}

	cancel, done := startEngine(t, baseSettings(), logger, w)
	defer func() { cancel(); <-done }()

	require.Eventually(t, func() bool {
		for _, e := range observed.FilterMessage("retryable failure, nacked").All() {
			cm := e.ContextMap()
			if cm[logging.MessageIDKey] == "a" && cm[logging.ReceiveCountKey] == int64(1) {
				return true
			}
		}
		return false
	}, eventually, tick)

	// 再配送(retryable Nack)で ReceiveCount=2 の失敗ログも生じ、並行処理下ではログ順序が
	// 保証されないため index を固定せず、初回配送(ReceiveCount=1)のログを ID で特定して検証する。
	var target map[string]any
	for _, e := range observed.FilterMessage("retryable failure, nacked").All() {
		cm := e.ContextMap()
		if cm[logging.MessageIDKey] == "a" && cm[logging.ReceiveCountKey] == int64(1) {
			target = cm
			break
		}
	}
	require.NotNil(t, target, "初回配送(ReceiveCount=1)の失敗ログが存在すること")
	assert.Equal(t, "a", target[logging.MessageIDKey])
	assert.Equal(t, int64(1), target[logging.ReceiveCountKey])
}

// --- TestEngine_Run 異常系ヘルパー ---

func engineRunUnknownWorkerError(t *testing.T) {
	t.Parallel()

	w := testWorker{name: "known", cons: testkit.NewFake(), handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
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
}

func engineRunReceiveFailureContinuesPoll(t *testing.T) {
	t.Parallel()

	// Receive エラー（ctx 起因でない）はエラーとして記録・CB に計上され、loop は次 poll へ続く。
	f := testkit.NewFake()
	f.FailReceiveOnce(xerrors.New("broker unreachable"))
	f.Enqueue(bw.Message{ID: "a"})
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}

	cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
	defer func() { cancel(); <-done }()

	require.Eventually(t, func() bool { return len(f.AckedIDs()) == 1 }, eventually, tick)
}

func engineRunRetryableNacked(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
	f.Enqueue(bw.Message{ID: "a"})
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
		return xerrors.Wrap(apperror.ErrRetryable, "downstream")
	})}

	cancel, done := startEngine(t, baseSettings(), logging.NewTestLogger(t), w)
	defer func() { cancel(); <-done }()

	require.Eventually(t, func() bool { return len(f.NackedIDs()) >= 1 }, eventually, tick)
	assert.Empty(t, f.AckedIDs())
	// retryable は per-message backoff つきで再配送される。
	assert.True(t, f.NackBackoffApplied("a"), "NackWithBackoff で再配送されること")
}

func engineRunAckErrorLogged(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mc := mock_worker.NewMockConsumer(ctrl)
	mc.EXPECT().Ack(gomock.Any(), gomock.Any()).Return(xerrors.New("ack boom"))

	logger, observed := logging.NewObservedTestLogger(t)
	w := testWorker{name: "w", cons: mc, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
	eng, err := New([]bw.Worker{w}, baseSettings(), observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), logger)
	require.NoError(t, err)
	r := newRun(eng, w)

	r.ack(context.Background(), bw.Message{ID: "a"})

	assert.Equal(t, 1, observed.FilterMessage("ack error").Len())
}

func engineRunNackErrorLogged(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mc := mock_worker.NewMockConsumer(ctrl)
	mc.EXPECT().NackWithBackoff(gomock.Any(), gomock.Any(), gomock.Any()).Return(xerrors.New("nack boom"))

	logger, observed := logging.NewObservedTestLogger(t)
	w := testWorker{name: "w", cons: mc, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
	eng, err := New([]bw.Worker{w}, baseSettings(), observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), logger)
	require.NoError(t, err)
	r := newRun(eng, w)

	r.nack(context.Background(), bw.Message{ID: "a", ReceiveCount: 1})

	assert.Equal(t, 1, observed.FilterMessage("nack error").Len())
}

func engineRunExtendErrorHeartbeatContinues(t *testing.T) {
	t.Parallel()

	// Extend 失敗は握り潰さず log+metric で可視化しつつ、ハートビート goroutine は
	// 生き続け、handler 完了で Ack される（lease 延長失敗は致命ではない）。
	f := testkit.NewFake()
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

	// ハートビート goroutine は stop 後に "extend failed" を 1 度だけ出しうるため、
	// test 終了と競合しない t 非依存の logger を使う（このテストはログを検証しない）。
	logger, _ := logging.NewObservedTestLogger(t)
	cancel, done := startEngine(t, set, logger, w)
	defer func() { closeRelease(); cancel(); <-done }()

	require.Eventually(t, func() bool { return f.ExtendCount("a") >= 2 }, eventually, tick)
	closeRelease()
	require.Eventually(t, func() bool { return len(f.AckedIDs()) >= 1 }, eventually, tick)
}

func engineRunExtendErrorAfterStopSuppressed(t *testing.T) {
	t.Parallel()

	// Extend が ctx キャンセル後に失敗した場合は再配送されるため、warn ログや ExtendError metric を出さない。
	ctrl := gomock.NewController(t)
	mc := mock_worker.NewMockConsumer(ctrl)
	entered := make(chan struct{})
	mc.EXPECT().Extend(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ bw.Message, _ time.Duration) error {
			close(entered)
			<-ctx.Done() // 停止指示が来るまで待ってから失敗させる
			return xerrors.New("extend boom")
		})

	logger, observed := logging.NewObservedTestLogger(t)
	set := baseSettings()
	set.ExtendInterval = 2 * time.Millisecond
	w := testWorker{name: "w", cons: mc, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
	eng, err := New([]bw.Worker{w}, set, observability.NewNoopTracerFactory(t), observability.NewNoopWorkerMetrics(t), logger)
	require.NoError(t, err)
	r := newRun(eng, w)

	ctx, cancel := context.WithCancel(context.Background())
	stop := r.startHeartbeat(ctx, bw.Message{ID: "a"})
	<-entered // ticker 発火で Extend 呼び出し済み・ctx はまだ生存
	cancel()  // 停止 → Extend が失敗するが ctx.Err() 済みなので握り潰される
	stop()

	assert.Equal(t, 0, observed.FilterMessage("extend failed").Len())
}

func engineRunAcquireInterruptedByCancel(t *testing.T) {
	t.Parallel()

	f := testkit.NewFake()
	w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
	set := baseSettings()
	set.CircuitFailureThreshold = 1
	set.CircuitOpenBackoffInitial = time.Hour // cooldown より先に ctx がキャンセルされるよう十分長くする
	set.CircuitOpenBackoffMax = time.Hour
	r := newRun(newTestEngine(t, set, w), w)

	r.cb.onFailure()
	require.Equal(t, phaseOpen, r.cb.phaseNow())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	n, ok := r.acquire(ctx)

	assert.Equal(t, 0, n)
	assert.False(t, ok)
}

func TestEngine_drain(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx cancel 後も in-flight は drain 期限内に完了して Ack される", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
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

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DrainTimeout を超えて完了しない in-flight は待たずに抜け Ack されない", func(t *testing.T) {
			t.Parallel()

			f := testkit.NewFake()
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
			set := baseSettings()
			set.DrainTimeout = 20 * time.Millisecond

			r := newRun(newTestEngine(t, set, w), w)
			r.wg.Add(1) // 完了しない in-flight を模す

			start := time.Now()
			r.drain()
			elapsed := time.Since(start)
			r.wg.Done() // drain 用 goroutine をリークさせないため後始末

			assert.GreaterOrEqual(t, elapsed, set.DrainTimeout)
			assert.Less(t, elapsed, eventually)
			assert.Empty(t, f.AckedIDs())
		})
	})
}
