package dbslot

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	mock_dbslot "go-boilerplate/internal/cli/dbslot/mock"
	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("now が nil でも time.Now にフォールバックして生成できる", func(t *testing.T) {
			t.Parallel()

			r := NewRegistry(t.TempDir(), "/w/a", "b", time.Minute, 4, nil)
			require.NotNil(t, r)
			assert.Equal(t, 4, r.MaxSlots())
		})
	})
}

func TestRegistry_EnsureDir(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("通常ディレクトリを 0700 で用意する", func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "pool")
			r := NewRegistry(dir, "/w/a", "b", time.Minute, 4, nil)
			require.NoError(t, r.EnsureDir())

			fi, err := os.Stat(dir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o700), fi.Mode().Perm())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("symlink を向いた POOL_DIR は先読み攻撃対策として拒否する", func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			target := filepath.Join(base, "target")
			require.NoError(t, os.Mkdir(target, 0o700))
			link := filepath.Join(base, "link")
			require.NoError(t, os.Symlink(target, link))

			r := NewRegistry(link, "/w/a", "b", time.Minute, 4, nil)
			err := r.EnsureDir()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "symlink")
		})
	})
}

func TestRegistry_WriteMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("meta を 0600 で書き出す", func(t *testing.T) {
			t.Parallel()

			r := newTestRegistry(t, "/w/a", time.Unix(1000, 0))
			require.True(t, r.TryAcquireFresh(1))
			require.NoError(t, r.WriteMeta(1))

			fi, err := os.Stat(r.metaPath(1))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロックディレクトリが無ければ書き込みエラーを返す", func(t *testing.T) {
			t.Parallel()

			// TryAcquireFresh を呼ばず lock dir が無い状態で WriteMeta → 親が無く WriteFile 失敗。
			r := newTestRegistry(t, "/w/a", time.Unix(1000, 0))
			require.Error(t, r.WriteMeta(9))
		})
	})
}

func TestRegistry_ReadMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き出した owner / heartbeat を読み戻す", func(t *testing.T) {
			t.Parallel()

			now := time.Unix(12_345, 0)
			r := newTestRegistry(t, "/w/owner", now)
			require.True(t, r.TryAcquireFresh(1))
			require.NoError(t, r.WriteMeta(1))

			m, ok := r.ReadMeta(1)
			require.True(t, ok)
			assert.Equal(t, "/w/owner", m.Owner)
			assert.Equal(t, now.Unix(), m.Heartbeat.Unix())
		})
	})
}

func TestRegistry_AgeSeconds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("heartbeat からの経過秒を返す", func(t *testing.T) {
			t.Parallel()

			now := time.Unix(10_000, 0)
			r := newTestRegistry(t, "/w/a", now)
			require.True(t, r.TryAcquireFresh(1))
			require.NoError(t, r.WriteMeta(1))

			later := NewRegistry(r.dir, "/w/a", "b", time.Minute, 8, func() time.Time { return now.Add(5 * time.Second) })
			assert.Equal(t, int64(5), later.AgeSeconds(1))
		})

		t.Run("meta が無ければ -1 を返す", func(t *testing.T) {
			t.Parallel()

			r := newTestRegistry(t, "/w/a", time.Unix(1000, 0))
			assert.Equal(t, int64(-1), r.AgeSeconds(3))
		})
	})
}

func TestRegistry_Lock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得・解放後に再取得できる", func(t *testing.T) {
			t.Parallel()

			r := newTestRegistry(t, "/w/a", time.Unix(1000, 0))
			unlock, err := r.Lock()
			require.NoError(t, err)
			unlock()

			unlock2, err := r.Lock()
			require.NoError(t, err)
			unlock2()
		})
	})
}

func TestPool_ensureLocalEnv(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local 系 env と未設定は許可する", func(t *testing.T) {
			t.Parallel()

			for _, env := range []string{config.EnvLocal, config.EnvCI, config.EnvTest, ""} {
				p, _, _ := newMockPool(t, t.TempDir(), env)
				assert.NoError(t, p.ensureLocalEnv())
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deploy 系 env は拒否する", func(t *testing.T) {
			t.Parallel()

			for _, env := range []string{config.EnvDevelopment, config.EnvStaging, config.EnvProduction} {
				p, _, _ := newMockPool(t, t.TempDir(), env)
				assert.Error(t, p.ensureLocalEnv())
			}
		})
	})
}

func TestPool_heldSlot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run(".gobp-db-slot の SLOT を返す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, ".gobp-db-slot"), []byte("SLOT=3\nDB_NAME_TEST=wt3_test\n"), 0o600))
			p, _, _ := newMockPool(t, root, "")

			slot, ok := p.heldSlot()
			assert.True(t, ok)
			assert.Equal(t, 3, slot)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run(".gobp-db-slot が無ければ false", func(t *testing.T) {
			t.Parallel()

			p, _, _ := newMockPool(t, t.TempDir(), "")
			_, ok := p.heldSlot()
			assert.False(t, ok)
		})
	})
}

func TestPool_Heartbeat(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持スロットの heartbeat を更新する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, comp := newMockPool(t, root, "")
			comp.EXPECT().UpSharedDB(gomock.Any(), "gobp-shared").Return(nil)
			expectSlotDBs(admin, "1")
			require.NoError(t, pool.Acquire(context.Background()))

			// heartbeat 更新でエラーにならず、meta が読める状態を維持する。
			require.NoError(t, pool.Heartbeat())
			_, ok := pool.reg.ReadMeta(1)
			assert.True(t, ok)
		})

		t.Run("スロット未保持なら何もしない", func(t *testing.T) {
			t.Parallel()

			pool, _, _ := newMockPool(t, t.TempDir(), "")
			require.NoError(t, pool.Heartbeat())
		})
	})
}

func TestPool_Status(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダと全スロット行を出力する", func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			ctrl := gomock.NewController(t)
			reg := NewRegistry(t.TempDir(), "/w/a", "b", 30*time.Minute, 8, func() time.Time { return time.Unix(1000, 0) })
			cfg := Config{Root: "/w/a", SharedProject: "gobp-shared", APIBasePort: 8080, MockAuthBase: 4000}
			p := NewPool(reg, mock_dbslot.NewMockDBAdmin(ctrl), mock_dbslot.NewMockCompose(ctrl), cfg, &out, &out)

			require.NoError(t, p.Status())
			s := out.String()
			assert.Contains(t, s, "SLOT")
			assert.Contains(t, s, "wt1_local")
			assert.Contains(t, s, "free")
		})

		t.Run("保持中スロットは in-use / owner を表示する", func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			ctrl := gomock.NewController(t)
			reg := NewRegistry(t.TempDir(), "/w/self", "b", 30*time.Minute, 8, func() time.Time { return time.Unix(1_000_000, 0) })
			require.True(t, reg.TryAcquireFresh(1))
			require.NoError(t, reg.WriteMeta(1))
			cfg := Config{Root: "/w/self", SharedProject: "gobp-shared", APIBasePort: 8080, MockAuthBase: 4000}
			p := NewPool(reg, mock_dbslot.NewMockDBAdmin(ctrl), mock_dbslot.NewMockCompose(ctrl), cfg, &out, &out)

			require.NoError(t, p.Status())
			s := out.String()
			assert.Contains(t, s, "in-use")
			assert.Contains(t, s, "/w/self")
		})

		t.Run("TTL 超過の他 owner リースは stale を表示する", func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			ctrl := gomock.NewController(t)
			reg := NewRegistry(t.TempDir(), "/w/me", "b", 30*time.Minute, 8, func() time.Time { return time.Unix(1_000_000, 0) })
			// 別 owner が古い heartbeat で保持中 → 自分から見て stale。
			other := NewRegistry(reg.dir, "/w/other", "b", 30*time.Minute, 8, func() time.Time { return time.Unix(1, 0) })
			require.True(t, other.TryAcquireFresh(1))
			require.NoError(t, other.WriteMeta(1))
			cfg := Config{Root: "/w/me", SharedProject: "gobp-shared", APIBasePort: 8080, MockAuthBase: 4000}
			p := NewPool(reg, mock_dbslot.NewMockDBAdmin(ctrl), mock_dbslot.NewMockCompose(ctrl), cfg, &out, &out)

			require.NoError(t, p.Status())
			assert.Contains(t, out.String(), "stale")
		})
	})
}

// newCapturingPool は、標準出力を捕捉できる Pool と DBAdmin mock・出力バッファを返す。
func newCapturingPool(t *testing.T, root string) (*Pool, *mock_dbslot.MockDBAdmin, *bytes.Buffer) {
	t.Helper()
	ctrl := gomock.NewController(t)
	admin := mock_dbslot.NewMockDBAdmin(ctrl)
	reg := NewRegistry(t.TempDir(), root, "branch", 30*time.Minute, 8, func() time.Time { return time.Unix(1_000_000, 0) })
	cfg := Config{
		Root: root, SharedProject: "gobp-shared",
		APIBasePort: 8080, MockAuthBase: 4000, DlvBase: 2345, PprofBase: 6060,
	}
	var out bytes.Buffer
	return NewPool(reg, admin, mock_dbslot.NewMockCompose(ctrl), cfg, &out, io.Discard), admin, &out
}

func TestPool_finishAcquire(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("meta 記録・DB 用意・スロットファイル書き出し・標準出力までを完了する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, out := newCapturingPool(t, root)
			require.NoError(t, pool.reg.EnsureDir())
			require.True(t, pool.reg.TryAcquireFresh(3)) // finishAcquire はリース取得済みのスロットに対して呼ばれる
			expectSlotDBs(admin, "3")

			require.NoError(t, pool.finishAcquire(context.Background(), 3, "acquired"))

			m, ok := pool.reg.ReadMeta(3)
			require.True(t, ok)
			assert.Equal(t, root, m.Owner)
			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "SLOT=3")
			// make は .gobp-db-slot を -include で読む。標準出力は人が確認するためのエコー。
			assert.Contains(t, out.String(), "SLOT=3")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロット未取得のまま呼ばれた場合は meta 書き込みに失敗しエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, _, _ := newCapturingPool(t, root)
			require.NoError(t, pool.reg.EnsureDir())

			require.Error(t, pool.finishAcquire(context.Background(), 3, "acquired"))

			_, err := os.Stat(filepath.Join(root, ".gobp-db-slot"))
			assert.ErrorIs(t, err, os.ErrNotExist)
		})

		t.Run("DB 用意に失敗した場合はスロットファイルを書かずエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, admin, _ := newCapturingPool(t, root)
			require.NoError(t, pool.reg.EnsureDir())
			require.True(t, pool.reg.TryAcquireFresh(3))
			admin.EXPECT().EnsureDatabase(gomock.Any(), "wt3_local").Return(errBoom)

			require.ErrorIs(t, pool.finishAcquire(context.Background(), 3, "acquired"), errBoom)

			_, err := os.Stat(filepath.Join(root, ".gobp-db-slot"))
			assert.ErrorIs(t, err, os.ErrNotExist)
		})

		t.Run("スロットファイルの書き出しに失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, _ := newCapturingPool(t, filepath.Join(t.TempDir(), "missing"))
			require.NoError(t, pool.reg.EnsureDir())
			require.True(t, pool.reg.TryAcquireFresh(3))
			expectSlotDBs(admin, "3")

			require.Error(t, pool.finishAcquire(context.Background(), 3, "acquired"))
		})
	})
}

func TestPool_ensureSlotDBs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local / test の両 DB を作成し設定まで行う", func(t *testing.T) {
			t.Parallel()

			pool, admin, _ := newCapturingPool(t, t.TempDir())
			expectSlotDBs(admin, "2")

			require.NoError(t, pool.ensureSlotDBs(context.Background(), 2))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB 作成に失敗した場合は設定へ進まずエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, _ := newCapturingPool(t, t.TempDir())
			admin.EXPECT().EnsureDatabase(gomock.Any(), "wt2_local").Return(errBoom)

			require.ErrorIs(t, pool.ensureSlotDBs(context.Background(), 2), errBoom)
		})

		t.Run("DB 設定に失敗した場合は次の DB へ進まずエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, admin, _ := newCapturingPool(t, t.TempDir())
			admin.EXPECT().EnsureDatabase(gomock.Any(), "wt2_local").Return(nil)
			admin.EXPECT().SetupDatabase(gomock.Any(), "wt2_local").Return(errBoom)

			require.ErrorIs(t, pool.ensureSlotDBs(context.Background(), 2), errBoom)
		})
	})
}

func TestPool_writeSlotFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB 名とホスト公開ポートをスロット番号で相対化して書き出す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, _, _ := newCapturingPool(t, root)

			require.NoError(t, pool.writeSlotFile(5))

			path := filepath.Join(root, ".gobp-db-slot")
			b, err := os.ReadFile(path) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			content := string(b)
			assert.Contains(t, content, "SLOT=5")
			assert.Contains(t, content, "DB_NAME_LOCAL=wt5_local")
			assert.Contains(t, content, "DB_NAME_TEST=wt5_test")
			assert.Contains(t, content, "API_HOST_PORT=8085")
			assert.Contains(t, content, "MOCK_AUTH_HOST_PORT=4005")
			assert.Contains(t, content, "DLV_HOST_PORT=2350")
			assert.Contains(t, content, "PPROF_HOST_PORT=6065")
			assert.Contains(t, content, "COMPOSE_PROJECT_NAME=gobp-shared")
			assert.Contains(t, content, "SERVE_PROJECT=gobp-wt-5")

			fi, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(filePerm), fi.Mode().Perm())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("出力先ディレクトリが存在しない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			pool, _, _ := newCapturingPool(t, filepath.Join(t.TempDir(), "missing"))

			require.Error(t, pool.writeSlotFile(1))
		})
	})
}

func TestPool_printSlotFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットファイルの内容をそのまま標準出力へ書き出す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			pool, _, out := newCapturingPool(t, root)
			require.NoError(t, pool.writeSlotFile(1))

			require.NoError(t, pool.printSlotFile())

			b, err := os.ReadFile(filepath.Join(root, ".gobp-db-slot")) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Equal(t, string(b), out.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットファイルが無い場合はエラーを返し何も出力しない", func(t *testing.T) {
			t.Parallel()

			pool, _, out := newCapturingPool(t, t.TempDir())

			require.Error(t, pool.printSlotFile())
			assert.Empty(t, out.String())
		})
	})
}
