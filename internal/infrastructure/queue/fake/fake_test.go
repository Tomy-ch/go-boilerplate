package fake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/xerrors"
)

func Test_Fake_Receive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("投入済みメッセージを最大 max 件取得し ReceiveCount=1 で in-flight へ移る", func(t *testing.T) {
			t.Parallel()

			f := New()
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

			f := New()
			f.Enqueue(worker.Message{ID: "a"})

			got, err := f.Receive(context.Background(), 10)

			require.NoError(t, err)
			assert.Len(t, got, 1)
			assert.Equal(t, 0, f.QueueLen())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入されたエラーを返す", func(t *testing.T) {
			t.Parallel()

			f := New()
			injected := xerrors.New("boom")
			f.FailReceiveOnce(injected)

			got, err := f.Receive(context.Background(), 1)

			require.ErrorIs(t, err, injected)
			assert.Nil(t, got)
		})

		t.Run("キューが空で ctx がキャンセルされた場合は ctx エラーを返す", func(t *testing.T) {
			t.Parallel()

			f := New()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := f.Receive(ctx, 1)

			require.ErrorIs(t, err, context.Canceled)
			assert.Nil(t, got)
		})
	})
}

func Test_Fake_Ack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Ack で in-flight から除去され記録される", func(t *testing.T) {
			t.Parallel()

			f := New()
			f.Enqueue(worker.Message{ID: "a"})
			got, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)

			require.NoError(t, f.Ack(context.Background(), got[0]))

			assert.Equal(t, []string{"a"}, f.AckedIDs())
			assert.Equal(t, 0, f.InflightLen())
		})
	})
}

func Test_Fake_Nack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Nack で再配送され、次の Receive で ReceiveCount が増える", func(t *testing.T) {
			t.Parallel()

			f := New()
			f.Enqueue(worker.Message{ID: "a"})
			first, err := f.Receive(context.Background(), 1)
			require.NoError(t, err)
			require.NoError(t, f.Nack(context.Background(), first[0]))

			second, err := f.Receive(context.Background(), 1)

			require.NoError(t, err)
			assert.Equal(t, []string{"a"}, f.NackedIDs())
			assert.Equal(t, "a", second[0].ID)
			assert.Equal(t, 2, second[0].ReceiveCount)
		})
	})
}

func Test_Fake_Extend(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Extend の呼び出し回数が記録される", func(t *testing.T) {
			t.Parallel()

			f := New()
			m := worker.Message{ID: "a"}

			require.NoError(t, f.Extend(context.Background(), m, 0))
			require.NoError(t, f.Extend(context.Background(), m, 0))

			assert.Equal(t, 2, f.ExtendCount("a"))
		})
	})
}

func Test_Fake_Fail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Fail の記録に message と cause が残る", func(t *testing.T) {
			t.Parallel()

			f := New()
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
