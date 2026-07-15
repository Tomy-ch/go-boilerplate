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
