package job

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewState(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilなStateを構築する", func(t *testing.T) {
			t.Parallel()

			assert.NotNil(t, NewState())
		})
	})
}

func Test_state_Set(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したジョブ名・引数・doneがSnapshotで取得できる", func(t *testing.T) {
			t.Parallel()

			s := NewState()
			doneCh := make(chan error, 1)

			s.Set("my-job", []string{"a", "b"}, doneCh)

			gotName, gotArgs, gotDone := s.Snapshot()
			assert.Equal(t, "my-job", gotName)
			assert.Equal(t, []string{"a", "b"}, gotArgs)
			require.Equal(t, doneCh, gotDone)

			// 別チャネルへ差し替わっていないこと（渡した done がそのまま使える）を確認する。
			gotDone <- nil
			assert.Len(t, doneCh, 1)
		})

		t.Run("複数回呼ぶと最後に設定した値で上書きされる", func(t *testing.T) {
			t.Parallel()

			s := NewState()

			s.Set("first", []string{"1"}, nil)
			s.Set("second", []string{"2"}, nil)

			gotName, gotArgs, _ := s.Snapshot()
			assert.Equal(t, "second", gotName)
			assert.Equal(t, []string{"2"}, gotArgs)
		})
	})
}

func Test_state_Snapshot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set前はゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			gotName, gotArgs, gotDone := NewState().Snapshot()

			assert.Empty(t, gotName)
			assert.Nil(t, gotArgs)
			assert.Nil(t, gotDone)
		})

		t.Run("Setと並行に呼んでもデータ競合せず設定後の値を返す", func(t *testing.T) {
			t.Parallel()

			s := NewState()

			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); s.Set("job", []string{"a"}, make(chan error, 1)) }()
			go func() { defer wg.Done(); _, _, _ = s.Snapshot() }()
			wg.Wait()

			gotName, gotArgs, gotDone := s.Snapshot()
			assert.Equal(t, "job", gotName)
			assert.Equal(t, []string{"a"}, gotArgs)
			assert.NotNil(t, gotDone)
		})
	})
}
