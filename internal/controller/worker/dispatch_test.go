package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bw "go-boilerplate/internal/usecase/boundary/worker"
)

func Test_newKeyedDispatcher(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("runner を 1 つも持たない状態で生成される", func(t *testing.T) {
			t.Parallel()

			kd := newKeyedDispatcher(3, func(context.Context, bw.Message) {})

			assert.NotNil(t, kd.runners)
			assert.Empty(t, kd.runners)
			assert.Equal(t, 3, kd.buffer)
		})

		t.Run("引数の buffer が key ごとのキュー長になる", func(t *testing.T) {
			t.Parallel()

			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			kd := newKeyedDispatcher(3, func(context.Context, bw.Message) {
				entered <- struct{}{}
				<-release
			})

			kd.dispatch(context.Background(), bw.Message{ID: "a", PartitionKey: "k"})
			select {
			case <-entered:
			case <-time.After(eventually):
				t.Fatal("key ごとの runner が起動しなかった")
			}

			kd.mu.Lock()
			r := kd.runners["k"]
			kd.mu.Unlock()

			require.NotNil(t, r)
			assert.Equal(t, 3, cap(r.ch))

			close(release)
		})
	})
}
