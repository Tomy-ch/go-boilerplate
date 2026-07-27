package dbslot

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mock_dbslot "go-boilerplate/internal/cli/dbslot/mock"
	"go-boilerplate/internal/config"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errBoom = xerrors.New("boom")

// fillSlots は、全スロットを別 owner の非 stale リースで埋める（no-free-slot テスト用）。
func fillSlots(t *testing.T, dir string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		r := NewRegistry(dir, "/w/other", "b", 30*time.Minute, n, func() time.Time { return time.Unix(1_000_000, 0) })
		require.True(t, r.TryAcquireFresh(i))
		require.NoError(t, r.WriteMeta(i))
	}
}

func newMockPool(t *testing.T, root, appEnv string) (*Pool, *mock_dbslot.MockDBAdmin, *mock_dbslot.MockCompose) {
	t.Helper()
	ctrl := gomock.NewController(t)
	admin := mock_dbslot.NewMockDBAdmin(ctrl)
	comp := mock_dbslot.NewMockCompose(ctrl)
	reg := NewRegistry(t.TempDir(), root, "branch", 30*time.Minute, 8, func() time.Time { return time.Unix(1_000_000, 0) })
	cfg := Config{
		Root: root, SharedProject: "gobp-shared",
		APIBasePort: 8080, MockAuthBase: 4000, DlvBase: 2345, PprofBase: 6060,
		APPEnv: appEnv,
	}
	return NewPool(reg, admin, comp, cfg, io.Discard, io.Discard), admin, comp
}

// expectSlotDBs は、指定スロットの DB 作成/設定呼び出しを期待に登録する。
func expectSlotDBs(admin *mock_dbslot.MockDBAdmin, slot string) {
	admin.EXPECT().EnsureDatabase(gomock.Any(), "wt"+slot+"_local").Return(nil)
	admin.EXPECT().SetupDatabase(gomock.Any(), "wt"+slot+"_local").Return(nil)
	admin.EXPECT().EnsureDatabase(gomock.Any(), "wt"+slot+"_test").Return(nil)
	admin.EXPECT().SetupDatabase(gomock.Any(), "wt"+slot+"_test").Return(nil)
}

func TestPool_Acquire(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空きスロット1を取得し DB 作成と .gobp-db-slot 書き出しを行う", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			expectSlotDBs(admin, "1")

			require.NoError(t, pool.Acquire(context.Background()))

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			content := string(b)
			assert.Contains(t, content, "SLOT=1")
			assert.Contains(t, content, "DB_NAME_LOCAL=wt1_local")
			assert.Contains(t, content, "DB_NAME_TEST=wt1_test")
			assert.Contains(t, content, "API_HOST_PORT=8081")
			assert.Contains(t, content, "MOCK_AUTH_HOST_PORT=4001")
			assert.Contains(t, content, "DLV_HOST_PORT=2346")
			assert.Contains(t, content, "PPROF_HOST_PORT=6061")
			assert.Contains(t, content, "COMPOSE_PROJECT_NAME=gobp-shared")
			assert.Contains(t, content, "SERVE_PROJECT=gobp-wt-1")
		})

		t.Run("stale スロットに稼働中接続があれば破壊せず次スロットを取得する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			// slot1 は稼働中接続ありで skip、slot2 を取得する。
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(0, nil)
			admin.EXPECT().ActiveConnections(gomock.Any(), "wt1_local", "wt1_test").Return(1, nil)
			expectSlotDBs(admin, "2")

			// slot1 を別 owner の stale リースにする。
			other := NewRegistry(pool.reg.dir, "/w/other", "b", 30*time.Minute, 8,
				func() time.Time { return time.Unix(1, 0) })
			require.True(t, other.TryAcquireFresh(1))
			require.NoError(t, other.WriteMeta(1))

			require.NoError(t, pool.Acquire(context.Background()))

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "SLOT=2")
		})

		t.Run("stale スロットで app コンテナが稼働中なら破壊せず次スロットを取得する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			// 接続プールはアイドルで空になるため、接続数 0 でも serve 中なら slot1 を守る。
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(2, nil)
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-2").Return(0, nil).AnyTimes()
			expectSlotDBs(admin, "2")

			other := NewRegistry(pool.reg.dir, "/w/other", "b", 30*time.Minute, 8,
				func() time.Time { return time.Unix(1, 0) })
			require.True(t, other.TryAcquireFresh(1))
			require.NoError(t, other.WriteMeta(1))

			require.NoError(t, pool.Acquire(context.Background()))

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "SLOT=2")
		})

		t.Run("再取得は保持中のスロットを再利用する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil).Times(2)
			admin.EXPECT().EnsureDatabase(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			admin.EXPECT().SetupDatabase(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			require.NoError(t, pool.Acquire(context.Background()))
			require.NoError(t, pool.Acquire(context.Background())) // reuse slot1

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "SLOT=1")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deploy 系 env（prd）では実行を拒否する", func(t *testing.T) {
			t.Parallel()

			// deploy 系は EnsureDir/UpSharedDB より前に弾かれるため mock 呼び出しは発生しない。
			pool, _, _ := newMockPool(t, t.TempDir(), config.EnvProduction)

			err := pool.Acquire(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "refuses to run")
		})

		t.Run("共有 DB 起動に失敗するとエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, _, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().UpSharedDB(gomock.Any(), gomock.Any()).Return(errBoom)

			require.Error(t, pool.Acquire(context.Background()))
		})

		t.Run("DB 作成に失敗するとエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().UpSharedDB(gomock.Any(), gomock.Any()).Return(nil)
			admin.EXPECT().EnsureDatabase(gomock.Any(), gomock.Any()).Return(errBoom)

			require.Error(t, pool.Acquire(context.Background()))
		})

		t.Run("DB セットアップに失敗するとエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().UpSharedDB(gomock.Any(), gomock.Any()).Return(nil)
			admin.EXPECT().EnsureDatabase(gomock.Any(), "wt1_local").Return(nil)
			admin.EXPECT().SetupDatabase(gomock.Any(), "wt1_local").Return(errBoom)

			require.Error(t, pool.Acquire(context.Background()))
		})

		t.Run("全スロット使用中ならエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, _, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().UpSharedDB(gomock.Any(), gomock.Any()).Return(nil)
			fillSlots(t, pool.reg.dir, pool.reg.MaxSlots())

			err := pool.Acquire(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no free slot")
		})
	})
}

func TestPool_slotInUse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("app コンテナが稼働していれば使用中と判定する", func(t *testing.T) {
			t.Parallel()

			pool, _, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(1, nil)

			busy, err := pool.slotInUse(context.Background(), 1)

			require.NoError(t, err)
			assert.True(t, busy)
		})

		t.Run("コンテナが無くても DB 接続があれば使用中と判定する", func(t *testing.T) {
			t.Parallel()

			pool, admin, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(0, nil)
			admin.EXPECT().ActiveConnections(gomock.Any(), "wt1_local", "wt1_test").Return(2, nil)

			busy, err := pool.slotInUse(context.Background(), 1)

			require.NoError(t, err)
			assert.True(t, busy)
		})

		t.Run("コンテナも DB 接続も無ければ未使用と判定する", func(t *testing.T) {
			t.Parallel()

			pool, admin, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(0, nil)
			admin.EXPECT().ActiveConnections(gomock.Any(), "wt1_local", "wt1_test").Return(0, nil)

			busy, err := pool.slotInUse(context.Background(), 1)

			require.NoError(t, err)
			assert.False(t, busy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コンテナ確認が失敗すればエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, _, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(0, errBoom)

			_, err := pool.slotInUse(context.Background(), 1)

			require.ErrorIs(t, err, errBoom)
		})

		t.Run("接続確認が失敗すればエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, comp := newMockPool(t, t.TempDir(), "")
			comp.EXPECT().RunningContainers(gomock.Any(), "gobp-wt-1").Return(0, nil)
			admin.EXPECT().ActiveConnections(gomock.Any(), "wt1_local", "wt1_test").Return(0, errBoom)

			_, err := pool.slotInUse(context.Background(), 1)

			require.ErrorIs(t, err, errBoom)
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
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			expectSlotDBs(admin, "1")
			comp.EXPECT().DownServe(gomock.Any(), "gobp-wt-1").Return(nil)

			require.NoError(t, pool.Acquire(context.Background()))
			require.NoError(t, pool.Release(context.Background()))

			_, err := os.Stat(filepath.Join(root, ".gobp-db-slot"))
			assert.True(t, os.IsNotExist(err))
		})

		t.Run("スロット未保持なら何もしない", func(t *testing.T) {
			t.Parallel()

			pool, _, _ := newMockPool(t, t.TempDir(), "")
			require.NoError(t, pool.Release(context.Background()))
		})

		t.Run("serve 停止が失敗してもリースは解放しログへ可視化する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			ctrl := gomock.NewController(t)
			admin := mock_dbslot.NewMockDBAdmin(ctrl)
			comp := mock_dbslot.NewMockCompose(ctrl)
			reg := NewRegistry(t.TempDir(), root, "branch", 30*time.Minute, 8,
				func() time.Time { return time.Unix(1_000_000, 0) })
			cfg := Config{
				Root: root, SharedProject: "gobp-shared",
				APIBasePort: 8080, MockAuthBase: 4000, DlvBase: 2345, PprofBase: 6060,
			}
			var logbuf bytes.Buffer
			pool := NewPool(reg, admin, comp, cfg, io.Discard, &logbuf)

			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			expectSlotDBs(admin, "1")
			comp.EXPECT().DownServe(gomock.Any(), "gobp-wt-1").Return(errBoom)

			require.NoError(t, pool.Acquire(context.Background()))
			require.NoError(t, pool.Release(context.Background()))

			_, err := os.Stat(filepath.Join(root, ".gobp-db-slot"))
			assert.True(t, os.IsNotExist(err))
			assert.Contains(t, logbuf.String(), "failed to stop serve containers for slot 1")
		})
	})
}

func TestPool_slotFileHasNoTrailingSpace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pool, admin, comp := newMockPool(t, root, "")
	comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
	expectSlotDBs(admin, "1")
	require.NoError(t, pool.Acquire(context.Background()))

	b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
	require.NoError(t, err)
	for line := range strings.SplitSeq(strings.TrimRight(string(b), "\n"), "\n") {
		assert.Equal(t, line, strings.TrimSpace(line), "行に余分な空白がないこと")
	}
}
