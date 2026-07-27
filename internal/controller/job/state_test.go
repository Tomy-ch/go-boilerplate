package job

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestState(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SetとSnapshotは値を保存して返す", func(t *testing.T) {
			t.Parallel()
			s := NewState()

			name := "my-job"
			args := []string{"a", "b"}
			doneCh := make(chan error, 1)

			s.Set(name, args, doneCh)

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

		t.Run("Setを複数回呼ぶとSnapshotは最後の値を返す", func(t *testing.T) {
			t.Parallel()
			s := NewState()

			s.Set("first", []string{"1"}, nil)
			s.Set("second", []string{"2"}, nil)

			gotName, gotArgs, _ := s.Snapshot()
			assert.Equal(t, "second", gotName)
			assert.Equal(t, []string{"2"}, gotArgs)
		})

		t.Run("Set前のSnapshotはゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			gotName, gotArgs, gotDone := NewState().Snapshot()
			assert.Empty(t, gotName)
			assert.Nil(t, gotArgs)
			assert.Nil(t, gotDone)
		})

		t.Run("SetとSnapshotを並行に呼んでもデータ競合しない", func(t *testing.T) {
			t.Parallel()
			s := NewState()

			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); s.Set("job", []string{"a"}, make(chan error, 1)) }()
			go func() { defer wg.Done(); _, _, _ = s.Snapshot() }()
			wg.Wait()

			gotName, _, _ := s.Snapshot()
			assert.Equal(t, "job", gotName)
		})
	})
}

func Test_state_Set(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_state_Snapshot(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
