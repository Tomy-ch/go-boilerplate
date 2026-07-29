package testkit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

func TestFake_Receive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("投入済みメッセージを最大 max 件取得し ReceiveCount=1 で in-flight へ移る", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(
				worker.Message{ID: "a"},
				worker.Message{ID: "b"},
				worker.Message{ID: "c"},
			)

			got, err := f.Receive(context.Background(), 2)

			require.NoError(t, err)
			assert.Len(t, got, 2)
			assert.Equal(t, "a", got[0].ID)
			assert.Equal(t, 1, got[0].ReceiveCount)
			assert.Equal(t, 1, f.QueueLen())
			assert.Equal(t, 2, f.InflightLen())
		})

		t.Run("max がキュー長を上回る場合は残り全件を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})

			got, err := f.Receive(context.Background(), 10)

			require.NoError(t, err)
			assert.Len(t, got, 1)
			assert.Equal(t, 0, f.QueueLen())
			assert.Equal(t, 1, f.InflightLen())
		})

		t.Run("キューが空の状態で待機中の Receive が Enqueue の起床で取得できる", func(t *testing.T) {
			t.Parallel()

			f := NewFake()

			type result struct {
				msgs []worker.Message
				err  error
			}
			// Receive がバグで起床しなくても CI を無期限ハングさせないよう、待機側に
			// タイムアウト付き context を渡し、結果受信も select でタイムアウト失敗にする。
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			done := make(chan result, 1)
			go func() {
				msgs, err := f.Receive(ctx, 1)
				done <- result{msgs: msgs, err: err}
			}()

			f.Enqueue(worker.Message{ID: "a"})

			select {
			case got := <-done:
				require.NoError(t, got.err)
				assert.Len(t, got.msgs, 1)
				assert.Equal(t, "a", got.msgs[0].ID)
				assert.Equal(t, 1, got.msgs[0].ReceiveCount)
			case <-time.After(2 * time.Second):
				t.Fatal("Enqueue の起床で待機中の Receive が返らなかった")
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入されたエラーを返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			injected := xerrors.New("boom")
			f.FailReceiveOnce(injected)

			got, err := f.Receive(context.Background(), 1)

			require.ErrorIs(t, err, injected)
			assert.Nil(t, got)
		})

		t.Run("複数注入されたエラーを注入順に返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			first := xerrors.New("first")
			second := xerrors.New("second")
			f.FailReceiveOnce(first)
			f.FailReceiveOnce(second)

			got1, err1 := f.Receive(context.Background(), 1)
			require.ErrorIs(t, err1, first)
			assert.Nil(t, got1)

			got2, err2 := f.Receive(context.Background(), 1)
			require.ErrorIs(t, err2, second)
			assert.Nil(t, got2)
		})

		t.Run("キューが空で ctx がキャンセルされた場合は ctx エラーを返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := f.Receive(ctx, 1)

			require.ErrorIs(t, err, context.Canceled)
			assert.Nil(t, got)
		})
	})
}

func TestFake_Ack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Ack で in-flight から除去され記録される", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			got, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)

			require.NoError(t, f.Ack(context.Background(), got[0]))

			assert.Equal(t, []string{"a"}, f.AckedIDs())
			assert.Equal(t, 0, f.InflightLen())
		})
	})
}

func TestFake_Nack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Nack で再配送され、次の Receive で ReceiveCount が増える", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			first, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)
			require.NoError(t, f.Nack(context.Background(), first[0]))
			assert.Equal(t, 0, f.InflightLen())

			second, err := f.Receive(context.Background(), 1)

			require.NoError(t, err)
			assert.Equal(t, []string{"a"}, f.NackedIDs())
			assert.Equal(t, "a", second[0].ID)
			assert.Equal(t, 2, second[0].ReceiveCount)
		})
	})
}

func TestFake_NackWithBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("遅延を記録して再配送し、ReceiveCountが増える", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			first, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)
			require.NoError(t, f.NackWithBackoff(context.Background(), first[0], 5*time.Second))

			second, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)

			assert.Equal(t, []string{"a"}, f.NackedIDs())
			assert.Equal(t, 5*time.Second, f.NackBackoffOf("a"))
			assert.True(t, f.NackBackoffApplied("a"))
			assert.Equal(t, 2, second[0].ReceiveCount)
		})

		t.Run("NackWithBackoff未呼び出しのIDはAppliedがfalse", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			assert.False(t, f.NackBackoffApplied("missing"))
			assert.Equal(t, time.Duration(0), f.NackBackoffOf("missing"))
		})
	})
}

func TestFake_Extend(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Extend の呼び出し回数が記録される", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			m := worker.Message{ID: "a"}

			require.NoError(t, f.Extend(context.Background(), m, 0))
			require.NoError(t, f.Extend(context.Background(), m, 0))

			assert.Equal(t, 2, f.ExtendCount("a"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SetExtendErr で設定したエラーを返し呼び出し回数は記録される", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			m := worker.Message{ID: "a"}
			injected := xerrors.New("extend boom")
			f.SetExtendErr(injected)

			err := f.Extend(context.Background(), m, 0)

			require.ErrorIs(t, err, injected)
			assert.Equal(t, 1, f.ExtendCount("a"))
		})
	})
}

func TestFake_Fail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Fail の記録に message と cause が残る", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			m := worker.Message{ID: "a"}
			cause := xerrors.New("permanent reason")

			require.NoError(t, f.Fail(context.Background(), m, cause))

			failed := f.Failed()
			require.Len(t, failed, 1)
			assert.Equal(t, "a", failed[0].Message.ID)
			require.ErrorIs(t, failed[0].Cause, cause)
		})
	})
}

func TestFake_AckedIDs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Ack の呼び出し順に ID を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Ack(context.Background(), worker.Message{ID: "a"}))
			require.NoError(t, f.Ack(context.Background(), worker.Message{ID: "b"}))

			assert.Equal(t, []string{"a", "b"}, f.AckedIDs())
		})

		t.Run("返り値を書き換えても記録には影響しない", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Ack(context.Background(), worker.Message{ID: "a"}))

			got := f.AckedIDs()
			got[0] = "mutated"

			assert.Equal(t, []string{"a"}, f.AckedIDs())
		})
	})
}

func TestFake_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("投入した件数だけキューが伸び in-flight は増えない", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"}, worker.Message{ID: "b"})

			assert.Equal(t, 2, f.QueueLen())
			assert.Equal(t, 0, f.InflightLen())
		})

		t.Run("複数回の投入は末尾へ追加され投入順に取得できる", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			f.Enqueue(worker.Message{ID: "b"}, worker.Message{ID: "c"})

			got, err := f.Receive(context.Background(), 3)

			require.NoError(t, err)
			require.Len(t, got, 3)
			assert.Equal(t, "a", got[0].ID)
			assert.Equal(t, "b", got[1].ID)
			assert.Equal(t, "c", got[2].ID)
		})
	})
}

func TestFake_ExtendCount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ID ごとに独立して Extend の呼び出し回数を数える", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Extend(context.Background(), worker.Message{ID: "a"}, time.Second))
			require.NoError(t, f.Extend(context.Background(), worker.Message{ID: "a"}, time.Second))
			require.NoError(t, f.Extend(context.Background(), worker.Message{ID: "b"}, time.Second))

			assert.Equal(t, 2, f.ExtendCount("a"))
			assert.Equal(t, 1, f.ExtendCount("b"))
		})

		t.Run("Extend されていない ID は 0 を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()

			assert.Equal(t, 0, f.ExtendCount("missing"))
		})
	})
}

func TestFake_FailReceiveOnce(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入したエラーは 1 度だけ返り、以降の Receive はメッセージを返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			injected := xerrors.New("boom")
			f.FailReceiveOnce(injected)
			f.Enqueue(worker.Message{ID: "a"})

			_, err := f.Receive(context.Background(), 1)
			require.ErrorIs(t, err, injected)

			got, err := f.Receive(context.Background(), 1)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "a", got[0].ID)
		})
	})
}

func TestFake_Failed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Fail の呼び出し順に message と cause を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			firstCause := xerrors.New("first")
			secondCause := xerrors.New("second")
			require.NoError(t, f.Fail(context.Background(), worker.Message{ID: "a"}, firstCause))
			require.NoError(t, f.Fail(context.Background(), worker.Message{ID: "b"}, secondCause))

			failed := f.Failed()

			require.Len(t, failed, 2)
			assert.Equal(t, "a", failed[0].Message.ID)
			require.ErrorIs(t, failed[0].Cause, firstCause)
			assert.Equal(t, "b", failed[1].Message.ID)
			require.ErrorIs(t, failed[1].Cause, secondCause)
		})

		t.Run("返り値を書き換えても記録には影響しない", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Fail(context.Background(), worker.Message{ID: "a"}, xerrors.New("boom")))

			got := f.Failed()
			got[0].Message.ID = "mutated"

			assert.Equal(t, "a", f.Failed()[0].Message.ID)
		})
	})
}

func TestFake_InflightLen(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Receive で増え Ack で減る", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"}, worker.Message{ID: "b"})
			got, err := f.Receive(context.Background(), 2)
			require.NoError(t, err)
			assert.Equal(t, 2, f.InflightLen())

			require.NoError(t, f.Ack(context.Background(), got[0]))

			assert.Equal(t, 1, f.InflightLen())
		})

		t.Run("Nack で再配送されたメッセージは in-flight から外れる", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			got, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)

			require.NoError(t, f.Nack(context.Background(), got[0]))

			assert.Equal(t, 0, f.InflightLen())
		})
	})
}

func TestFake_NackBackoffApplied(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("遅延 0 で呼ばれた場合も呼び出し有無として true を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.NackWithBackoff(context.Background(), worker.Message{ID: "a"}, 0))

			assert.True(t, f.NackBackoffApplied("a"))
			assert.Equal(t, time.Duration(0), f.NackBackoffOf("a"))
		})

		t.Run("Nack のみ呼ばれた ID は false を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Nack(context.Background(), worker.Message{ID: "a"}))

			assert.False(t, f.NackBackoffApplied("a"))
		})
	})
}

func TestFake_NackBackoffOf(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ ID で複数回要求された場合は最後の遅延を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			m := worker.Message{ID: "a"}
			require.NoError(t, f.NackWithBackoff(context.Background(), m, 5*time.Second))
			require.NoError(t, f.NackWithBackoff(context.Background(), m, 2*time.Second))

			assert.Equal(t, 2*time.Second, f.NackBackoffOf("a"))
		})

		t.Run("記録の無い ID は 0 を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()

			assert.Equal(t, time.Duration(0), f.NackBackoffOf("missing"))
		})
	})
}

func TestFake_NackedIDs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Nack の呼び出し順に ID を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Nack(context.Background(), worker.Message{ID: "a"}))
			require.NoError(t, f.Nack(context.Background(), worker.Message{ID: "b"}))

			assert.Equal(t, []string{"a", "b"}, f.NackedIDs())
		})

		t.Run("返り値を書き換えても記録には影響しない", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			require.NoError(t, f.Nack(context.Background(), worker.Message{ID: "a"}))

			got := f.NackedIDs()
			got[0] = "mutated"

			assert.Equal(t, []string{"a"}, f.NackedIDs())
		})
	})
}

func TestFake_QueueLen(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("投入で増え Receive した分だけ減る", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"}, worker.Message{ID: "b"}, worker.Message{ID: "c"})
			assert.Equal(t, 3, f.QueueLen())

			_, err := f.Receive(context.Background(), 2)

			require.NoError(t, err)
			assert.Equal(t, 1, f.QueueLen())
		})

		t.Run("Nack で戻されたメッセージは再びキューに数えられる", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"})
			got, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)
			require.Equal(t, 0, f.QueueLen())

			require.NoError(t, f.Nack(context.Background(), got[0]))

			assert.Equal(t, 1, f.QueueLen())
		})
	})
}

func TestFake_SetExtendErr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定後の Extend は毎回そのエラーを返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			injected := xerrors.New("extend boom")
			f.SetExtendErr(injected)

			require.ErrorIs(t, f.Extend(context.Background(), worker.Message{ID: "a"}, time.Second), injected)
			require.ErrorIs(t, f.Extend(context.Background(), worker.Message{ID: "a"}, time.Second), injected)
		})

		t.Run("nil を設定し直すと Extend は成功へ戻る", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.SetExtendErr(xerrors.New("extend boom"))
			f.SetExtendErr(nil)

			require.NoError(t, f.Extend(context.Background(), worker.Message{ID: "a"}, time.Second))
		})
	})
}

func TestFake_nackLocked(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("in-flight から外してキュー末尾へ戻し nacked に記録する", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			f.Enqueue(worker.Message{ID: "a"}, worker.Message{ID: "b"})
			got, err := f.Receive(context.Background(), 2)
			require.NoError(t, err)

			f.mu.Lock()
			f.nackLocked(got[0])
			f.mu.Unlock()

			assert.Equal(t, 1, f.InflightLen())
			assert.Equal(t, 1, f.QueueLen())
			assert.Equal(t, []string{"a"}, f.NackedIDs())
		})
	})
}

func TestFake_signal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待機用チャネルを閉じたうえで新しいチャネルへ差し替える", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			old := f.notify

			f.mu.Lock()
			f.signal()
			f.mu.Unlock()

			assert.True(t, isClosed(old), "旧チャネルが閉じられ待機中の Receive が起床する")
			assert.NotEqual(t, old, f.notify)
			assert.False(t, isClosed(f.notify), "差し替え後のチャネルは次の起床まで開いたまま")
		})

		t.Run("連続して呼んでも毎回差し替わるため閉じ済みチャネルを再度閉じない", func(t *testing.T) {
			t.Parallel()

			f := NewFake()

			f.mu.Lock()
			f.signal()
			f.signal()
			second := f.notify
			f.mu.Unlock()

			assert.False(t, isClosed(second))
		})
	})
}

// isClosed は、ブロックせずにチャネルが閉じられているかを判定します。
func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestNewFake(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キュー・in-flight・各記録が空の Fake を返す", func(t *testing.T) {
			t.Parallel()

			f := NewFake()

			assert.Equal(t, 0, f.QueueLen())
			assert.Equal(t, 0, f.InflightLen())
			assert.Empty(t, f.AckedIDs())
			assert.Empty(t, f.NackedIDs())
			assert.Empty(t, f.Failed())
		})

		t.Run("ID ごとの記録マップが初期化済みで生成直後から記録できる", func(t *testing.T) {
			t.Parallel()

			f := NewFake()
			m := worker.Message{ID: "a"}

			require.NoError(t, f.Extend(context.Background(), m, time.Second))
			require.NoError(t, f.NackWithBackoff(context.Background(), m, time.Second))

			assert.Equal(t, 1, f.ExtendCount("a"))
			assert.Equal(t, time.Second, f.NackBackoffOf("a"))
		})
	})
}
