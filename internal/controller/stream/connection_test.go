package stream

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/stream/gen"
	"go-boilerplate/internal/logging"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_ucrealtime "go-boilerplate/internal/usecase/realtime/mock"
	"go-boilerplate/pkg/xerrors"
)

// tick の待ちを識別する名前。fakeSleeper は要求された待ち時間からこれを引きます。
const (
	tickHeartbeat = "heartbeat"
	tickLifetime  = "lifetime"
	tickCatchUp   = "catchup"
)

// waitTimeout は、テストが合図を待つ上限です。超えたら実装が動いていないので落とします。
const waitTimeout = 3 * time.Second

var _ net.Error = timeoutError{}

// timeoutError は、Timeout() が true を返す net.Error です。
type timeoutError struct{}

// fakeSleeper は、待ち時間ごとにテストの合図を待つ Sleeper です。heartbeat / lifetime / catch-up は
// 待ち時間が違うので、テストは起こしたい tick だけを名指しで起こせます。実時間は 1 度も進みません。
type fakeSleeper struct {
	mu    sync.Mutex
	gates map[string]chan struct{}
}

func newFakeSleeper() *fakeSleeper {
	return &fakeSleeper{gates: map[string]chan struct{}{}}
}

// Sleep は、d が指す tick の合図が来るまで待ちます。
func (s *fakeSleeper) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-s.gate(tickNameOf(d)):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// tick は、name の待ちを 1 回だけ起こします。合図の受け手が居なければ落とします。
func (s *fakeSleeper) tick(t *testing.T, name string) {
	t.Helper()

	select {
	case s.gate(name) <- struct{}{}:
	case <-time.After(waitTimeout):
		t.Fatalf("%s の待ちが起きなかった", name)
	}
}

func (s *fakeSleeper) gate(name string) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.gates[name] == nil {
		s.gates[name] = make(chan struct{})
	}

	return s.gates[name]
}

// tickNameOf は、要求された待ち時間から tick の種類を引きます。catch-up は jitter が乗るので既定に落ちます。
func tickNameOf(d time.Duration) string {
	switch d {
	case heartbeatInterval:
		return tickHeartbeat
	case maxConnectionLifetime:
		return tickLifetime
	default:
		return tickCatchUp
	}
}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// testConn は、テスト用の接続を 1 本作ります。
func testConn() *connection {
	return newConnection(1, "subject-1", "stream-1")
}

// testFetcher は、mock の Replayer を持つ fetcher を作ります。枠は 1 本だけです。
func testFetcher(t *testing.T, conn *connection, from rt.Sequence) (*fetcher, *mock_ucrealtime.MockReplayer) {
	t.Helper()

	replayer := mock_ucrealtime.NewMockReplayer(gomock.NewController(t))

	return &fetcher{
		conn:     conn,
		replayer: replayer,
		sem:      semaphore.NewWeighted(1),
		sleeper:  newFakeSleeper(),
		log:      logging.NewTestLogger(t),
		fetched:  from,
	}, replayer
}

// testEvent は、seq の位置の最小限の封筒です。payload を持つ形は sseTestEvent が返します。
func testEvent(seq rt.Sequence) rt.DeliveryEvent {
	return rt.DeliveryEvent{
		EventID: "evt-" + seq.String(), StreamID: "stream-1", Sequence: seq,
		Type: "sample.thing.created.v1", OccurredAt: sseTestOccurredAt, SchemaVersion: 1,
	}
}

func Test_newConnection(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("索引に使う値を保持し buffer を replay の 1 ページ分だけ持つ", func(t *testing.T) {
			t.Parallel()

			conn := newConnection(7, "subject-1", "stream-1")

			assert.Equal(t, uint64(7), conn.id)
			assert.Equal(t, "subject-1", conn.subject)
			assert.Equal(t, rt.StreamID("stream-1"), conn.stream)
			assert.Equal(t, connectionBuffer, cap(conn.events))
		})
	})
}

func Test_connection_close(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("quit を閉じて理由を記録する", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.close(closeReasonRevoked)

			<-conn.quit
			assert.Equal(t, closeReasonRevoked, conn.reason)
		})

		t.Run("2 回目以降の理由は記録されない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.close(closeReasonRevoked)
			conn.close(closeReasonCanceled)

			assert.Equal(t, closeReasonRevoked, conn.reason, "最初に閉じると決めた理由が残ること")
		})
	})
}

func Test_connection_closeWith(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指示を積んでから閉じ、理由を記録する", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.closeWith(stopControl(), closeReasonRevoked)

			<-conn.quit
			assert.Equal(t, closeReasonRevoked, conn.reason)
			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, gen.STOP, got.Action)
		})

		t.Run("既に閉じている接続には何も積まず理由も変えない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.close(closeReasonCanceled)

			conn.closeWith(stopControl(), closeReasonRevoked)

			assert.Equal(t, closeReasonCanceled, conn.reason)
			_, ok := conn.takeControl()
			assert.False(t, ok)
		})
	})
}

func Test_connection_signalControl(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指示を 1 つ積む", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.signalControl(stopControl())

			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, gen.STOP, got.Action)
		})

		t.Run("既に積まれていれば後から来た指示は捨てる", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.signalControl(stopControl())
			conn.signalControl(reconnectControl())

			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, gen.STOP, got.Action, "先に積まれた指示が残ること")

			_, ok = conn.takeControl()
			assert.False(t, ok)
		})
	})
}

func Test_connection_takeControl(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("積まれていなければ待たずに false を返す", func(t *testing.T) {
			t.Parallel()

			_, ok := testConn().takeControl()

			assert.False(t, ok)
		})
	})
}

func Test_connection_wake(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending を進めて合図を送る", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.wake(9)

			assert.Equal(t, int64(9), conn.pending.Load())
			select {
			case <-conn.signal:
			default:
				t.Fatal("合図が送られていない")
			}
		})

		t.Run("後戻りする通知は pending を下げない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.wake(9)
			conn.wake(4)

			assert.Equal(t, int64(9), conn.pending.Load())
		})

		t.Run("重複した通知は合図 1 つへ畳まれ、送信側は詰まらない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			conn.wake(1)
			conn.wake(2)
			conn.wake(3)

			assert.Equal(t, int64(3), conn.pending.Load(), "batch を跨いだ通知が最大値へ畳まれること")

			// 合図は cap 1 で、溢れた分は捨てる。queue されると engine の loop が詰まる。
			<-conn.signal
			select {
			case <-conn.signal:
				t.Fatal("合図が queue されている")
			default:
			}
		})
	})
}

func Test_startTicker(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("sleeper が返るたびに tick を送る", func(t *testing.T) {
			t.Parallel()

			sleeper := newFakeSleeper()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			ch := startTicker(ctx, sleeper, func() time.Duration { return heartbeatInterval })
			sleeper.tick(t, tickHeartbeat)

			select {
			case <-ch:
			case <-time.After(waitTimeout):
				t.Fatal("tick が届かなかった")
			}
		})

		t.Run("ctx が終わると tick は止まる", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			ch := startTicker(ctx, newFakeSleeper(), func() time.Duration { return heartbeatInterval })
			cancel()

			select {
			case <-ch:
				t.Fatal("ctx 終了後に tick が届いた")
			case <-time.After(50 * time.Millisecond):
			}
		})
	})
}

func Test_jitteredCatchUp(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("固定周期以上、jitter の幅までに収まり、毎回同じ値にはならない", func(t *testing.T) {
			t.Parallel()

			seen := map[time.Duration]struct{}{}
			for range 50 {
				got := jitteredCatchUp()

				require.GreaterOrEqual(t, got, catchUpInterval)
				require.LessOrEqual(t, got, catchUpInterval+catchUpJitter)
				seen[got] = struct{}{}
			}

			// 揺らぎが消えると全接続が同時刻に EventLog を読みに行く。範囲だけでは退化を検出できない。
			assert.Greater(t, len(seen), 1, "50 標本がすべて同値になる確率は事実上ゼロ")
		})
	})
}

func Test_writeFailureReason(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限切れは追いつけない client として扱う", func(t *testing.T) {
			t.Parallel()

			got := writeFailureReason(xerrors.Wrap(timeoutError{}, "write sse frame"))

			assert.Equal(t, closeReasonSlowClient, got)
		})

		t.Run("それ以外は既に居ない client として扱う", func(t *testing.T) {
			t.Parallel()

			got := writeFailureReason(xerrors.Wrap(apperror.ErrUnavailable, "broken pipe"))

			assert.Equal(t, closeReasonClientGone, got)
		})
	})
}

func Test_fetcher_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回 replay を流し込んでから合図を待つ", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 7)
			require.NoError(t, f.sem.Acquire(context.Background(), 1))
			f.holdsSlot = true
			replayer.EXPECT().ReadPage(gomock.Any(), rt.StreamID("stream-1"), rt.Sequence(7)).
				Return([]rt.DeliveryEvent{testEvent(8)}, false, nil)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			go f.run(ctx)

			select {
			case got := <-conn.events:
				assert.Equal(t, rt.Sequence(8), got.Sequence)
			case <-time.After(waitTimeout):
				t.Fatal("初回 replay が届かなかった")
			}
		})

		t.Run("初回 replay を終えたら枠を返し、周期 catch-up で読み直して また返す", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 0)
			sleeper := newFakeSleeper()
			f.sleeper = sleeper
			require.NoError(t, f.sem.Acquire(context.Background(), 1))
			f.holdsSlot = true
			gomock.InOrder(
				replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(0)).
					Return([]rt.DeliveryEvent{testEvent(1)}, false, nil),
				replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(1)).
					Return([]rt.DeliveryEvent{testEvent(2)}, false, nil),
			)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			go f.run(ctx)

			require.Equal(t, rt.Sequence(1), (<-conn.events).Sequence)
			// 初回 replay の枠は読み終えた時点で返る（defer だけに退行すると寿命いっぱい握られる）。
			require.Eventually(t, func() bool { return f.sem.TryAcquire(1) }, waitTimeout, 5*time.Millisecond,
				"初回 replay 直後に枠が返ること")
			f.sem.Release(1)

			sleeper.tick(t, tickCatchUp)

			assert.Equal(t, rt.Sequence(2), (<-conn.events).Sequence)
			assert.Eventually(t, func() bool { return f.sem.TryAcquire(1) }, waitTimeout, 5*time.Millisecond,
				"catch-up が取った枠も返ること")
		})
	})
}

func Test_fetcher_awaitTrigger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("周期 catch-up は追いついていても読み直す", func(t *testing.T) {
			t.Parallel()

			f, _ := testFetcher(t, testConn(), 5)
			catchUp := make(chan struct{}, 1)
			catchUp <- struct{}{}

			assert.True(t, f.awaitTrigger(context.Background(), catchUp))
		})

		t.Run("追いついている wakeup では読み直さない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 5)
			conn.wake(5)

			// 追いついた合図では戻らないので、戻る契機は ctx の終了だけになる。
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			t.Cleanup(cancel)

			assert.False(t, f.awaitTrigger(ctx, nil))
		})

		t.Run("追いついていない wakeup では読み直す", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 5)
			conn.wake(6)

			assert.True(t, f.awaitTrigger(context.Background(), nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続が閉じられていれば false を返す", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 5)
			conn.close(closeReasonDraining)

			assert.False(t, f.awaitTrigger(context.Background(), nil))
		})
	})
}

func Test_fetcher_drainPages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("続きがある間は読み進める", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 0)
			gomock.InOrder(
				replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(0)).
					Return([]rt.DeliveryEvent{testEvent(1)}, true, nil),
				replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(1)).
					Return([]rt.DeliveryEvent{testEvent(2)}, false, nil),
			)

			f.drainPages(context.Background())

			assert.Equal(t, rt.Sequence(2), f.fetched)
			assert.Len(t, conn.events, 2)
		})

		t.Run("読むものが無ければ何もしない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 3)
			replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(3)).Return(nil, false, nil)

			f.drainPages(context.Background())

			assert.Equal(t, rt.Sequence(3), f.fetched)
			assert.Empty(t, conn.events)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EventLog が読めなくても接続は閉じない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 0)
			replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, false, xerrors.Wrap(apperror.ErrUnavailable, "store off"))

			f.drainPages(context.Background())

			select {
			case <-conn.quit:
				t.Fatal("依存の不調で接続を閉じてはいけない（回復が再接続の嵐になる）")
			default:
			}
		})

		t.Run("cursor の続きが無ければ RESYNC を積んで閉じる", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, replayer := testFetcher(t, conn, 3)
			replayer.EXPECT().ReadPage(gomock.Any(), gomock.Any(), rt.Sequence(3)).
				Return([]rt.DeliveryEvent{testEvent(9)}, false, nil)

			f.drainPages(context.Background())

			<-conn.quit
			assert.Equal(t, closeReasonResync, conn.reason)
			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, gen.RESYNC, got.Action)
			assert.Equal(t, gen.CURSORTOOOLD, got.Reason)
		})
	})
}

func Test_fetcher_push(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("buffer へ入れた分だけ位置が進む", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 0)

			assert.True(t, f.push([]rt.DeliveryEvent{testEvent(1), testEvent(2)}))
			assert.Equal(t, rt.Sequence(2), f.fetched)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("buffer が溢れたら RETRY_LATER を積んで閉じる", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 0)
			events := make([]rt.DeliveryEvent, 0, connectionBuffer+1)
			for i := range connectionBuffer + 1 {
				events = append(events, testEvent(rt.Sequence(i+1)))
			}

			assert.False(t, f.push(events))

			<-conn.quit
			assert.Equal(t, closeReasonSlowClient, conn.reason)
			got, ok := conn.takeControl()
			require.True(t, ok)
			assert.Equal(t, gen.RETRYLATER, got.Action)
		})

		t.Run("閉じられた接続へは入れない", func(t *testing.T) {
			t.Parallel()

			conn := testConn()
			f, _ := testFetcher(t, conn, 0)
			conn.close(closeReasonDraining)
			for range connectionBuffer {
				conn.events <- testEvent(1)
			}

			assert.False(t, f.push([]rt.DeliveryEvent{testEvent(1)}))
			assert.Equal(t, closeReasonDraining, conn.reason, "閉じた理由が上書きされないこと")
		})
	})
}

func Test_fetcher_releaseSlot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持していた枠を返し 2 度目は何もしない", func(t *testing.T) {
			t.Parallel()

			f, _ := testFetcher(t, testConn(), 0)
			require.NoError(t, f.sem.Acquire(context.Background(), 1))
			f.holdsSlot = true

			f.releaseSlot()
			f.releaseSlot()

			assert.True(t, f.sem.TryAcquire(1), "枠がちょうど 1 本返っていること")
		})
	})
}

func Test_reconnectControl(t *testing.T) {
	t.Parallel()

	got := reconnectControl()

	assert.Equal(t, gen.RECONNECT, got.Action)
	assert.Equal(t, gen.SERVERDRAINING, got.Reason)
}

func Test_retryLaterControl(t *testing.T) {
	t.Parallel()

	got := retryLaterControl()

	assert.Equal(t, gen.RETRYLATER, got.Action)
	assert.Equal(t, gen.TEMPORARILYOVERLOADED, got.Reason)
	require.NotNil(t, got.RetryAfterMs)
	assert.Equal(t, int(retryAfterHint.Milliseconds()), *got.RetryAfterMs, "ヘッダの秒とミリ秒で同じ目安を使うこと")
}

func Test_reauthenticateControl(t *testing.T) {
	t.Parallel()

	got := reauthenticateControl()

	assert.Equal(t, gen.REAUTHENTICATE, got.Action)
	assert.Equal(t, gen.AUTHREFRESHREQUIRED, got.Reason)
}

func Test_resyncControl(t *testing.T) {
	t.Parallel()

	got := resyncControl()

	assert.Equal(t, gen.RESYNC, got.Action)
	assert.Equal(t, gen.CURSORTOOOLD, got.Reason)
}

func Test_stopControl(t *testing.T) {
	t.Parallel()

	got := stopControl()

	assert.Equal(t, gen.STOP, got.Action)
	assert.Equal(t, gen.AUTHORIZATIONREVOKED, got.Reason)
}
