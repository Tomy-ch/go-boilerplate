package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState(t *testing.T) {
	t.Parallel()

	t.Run("NewStateは非nilを返す", func(t *testing.T) {
		t.Parallel()
		s := NewState()
		require.NotNil(t, s)
	})

	t.Run("SetとSnapshotは値を保存して返す", func(t *testing.T) {
		t.Parallel()
		s := NewState()

		done := make(chan error, 1)
		done <- nil
		// スナップショットが確実にチャネルを返せるようにする
		<-done

		name := "my-job"
		args := []string{"a", "b"}

		// Setメソッド用に新規作成済みチャンネルを作成する
		doneCh := make(chan error, 1)

		// Setメソッドを呼び出す
		s.Set(name, args, doneCh)

		// Snapshotは同じ値を返すことを確認する
		gotName, gotArgs, gotDone := s.Snapshot()
		assert.Equal(t, name, gotName)
		assert.Equal(t, args, gotArgs)
		assert.Equal(t, doneCh, gotDone)

		// チャネルが使用可能であることを確認する
		select {
		case gotDone <- nil:
		default:
			t.Fatalf("gotDone channel not usable")
		}
	})
}
