package dbpool

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdmin struct {
	ensured  []string
	setup    []string
	activeBy map[string]int
}

type fakeCompose struct {
	upProject    string
	downProjects []string
}

func (f *fakeAdmin) EnsureDatabase(_ context.Context, name string) error {
	f.ensured = append(f.ensured, name)
	return nil
}

func (f *fakeAdmin) SetupDatabase(_ context.Context, name string) error {
	f.setup = append(f.setup, name)
	return nil
}

func (f *fakeAdmin) ActiveConnections(_ context.Context, names ...string) (int, error) {
	sum := 0
	for _, n := range names {
		sum += f.activeBy[n]
	}
	return sum, nil
}

func (f *fakeCompose) UpSharedDB(_ context.Context, project string) error {
	f.upProject = project
	return nil
}

func (f *fakeCompose) DownServe(_ context.Context, project string) error {
	f.downProjects = append(f.downProjects, project)
	return nil
}

func newTestPool(t *testing.T, owner, appEnv string, admin DBAdmin, comp Compose) *Pool {
	t.Helper()
	reg := NewRegistry(t.TempDir(), owner, "branch", 30*time.Minute, 8, func() time.Time { return time.Unix(1000, 0) })
	cfg := Config{
		Root:          owner,
		SharedProject: "gobp-shared",
		APIBasePort:   8080,
		MockAuthBase:  4000,
		APPEnv:        appEnv,
	}
	return NewPool(reg, admin, comp, cfg, io.Discard, io.Discard)
}

func TestPool_Acquire(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空きスロット1を取得し DB 作成と .gobp-db-slot 書き出しを行う", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			admin := &fakeAdmin{activeBy: map[string]int{}}
			comp := &fakeCompose{}
			pool := newTestPool(t, root, "", admin, comp)

			require.NoError(t, pool.Acquire(context.Background()))

			assert.Equal(t, "gobp-shared", comp.upProject)
			assert.Equal(t, []string{"wt1_local", "wt1_test"}, admin.ensured)
			assert.Equal(t, []string{"wt1_local", "wt1_test"}, admin.setup)

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			content := string(b)
			assert.Contains(t, content, "SLOT=1")
			assert.Contains(t, content, "DB_NAME_LOCAL=wt1_local")
			assert.Contains(t, content, "DB_NAME_TEST=wt1_test")
			assert.Contains(t, content, "API_HOST_PORT=8081")
			assert.Contains(t, content, "COMPOSE_PROJECT_NAME=gobp-shared")
			assert.Contains(t, content, "SERVE_PROJECT=gobp-wt-1")
		})

		t.Run("stale スロットに稼働中接続があれば破壊せず次スロットを取得する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			// 他 worktree が slot1 を古い heartbeat で保持し、かつ wt1_local に接続が生きている状態。
			admin := &fakeAdmin{activeBy: map[string]int{"wt1_local": 1}}
			comp := &fakeCompose{}
			pool := newTestPool(t, root, "", admin, comp)

			// slot1 を別 owner の stale リースにする。
			other := NewRegistry(pool.reg.dir, "/w/other", "b", 30*time.Minute, 8,
				func() time.Time { return time.Unix(1, 0) })
			require.True(t, other.TryAcquireFresh(1))
			require.NoError(t, other.WriteMeta(1))

			require.NoError(t, pool.Acquire(context.Background()))

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			// slot1 は接続ありで skip され、slot2 が取得される。
			assert.Contains(t, string(b), "SLOT=2")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deploy 系 env（prd）では実行を拒否する", func(t *testing.T) {
			t.Parallel()

			pool := newTestPool(t, t.TempDir(), config.EnvProduction, &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})

			err := pool.Acquire(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "refuses to run")
		})
	})
}

func TestPool_Release(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("serve コンテナを停止し .gobp-db-slot を削除する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			admin := &fakeAdmin{activeBy: map[string]int{}}
			comp := &fakeCompose{}
			pool := newTestPool(t, root, "", admin, comp)
			require.NoError(t, pool.Acquire(context.Background()))

			require.NoError(t, pool.Release(context.Background()))

			assert.Equal(t, []string{"gobp-wt-1"}, comp.downProjects)
			_, err := os.Stat(filepath.Join(root, ".gobp-db-slot"))
			assert.True(t, os.IsNotExist(err))
		})
	})
}

func TestPool_slotFileHasNoTrailingSpace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pool := newTestPool(t, root, "", &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})
	require.NoError(t, pool.Acquire(context.Background()))

	b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
	require.NoError(t, err)
	for line := range strings.SplitSeq(strings.TrimRight(string(b), "\n"), "\n") {
		assert.Equal(t, line, strings.TrimSpace(line), "行に余分な空白がないこと")
	}
}
