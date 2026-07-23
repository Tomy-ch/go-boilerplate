package dbpool

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				p := newTestPool(t, t.TempDir(), env, &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})
				assert.NoError(t, p.ensureLocalEnv())
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deploy 系 env は拒否する", func(t *testing.T) {
			t.Parallel()

			for _, env := range []string{config.EnvDevelopment, config.EnvStaging, config.EnvProduction} {
				p := newTestPool(t, t.TempDir(), env, &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})
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
			p := newTestPool(t, root, "", &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})

			slot, ok := p.heldSlot()
			assert.True(t, ok)
			assert.Equal(t, 3, slot)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run(".gobp-db-slot が無ければ false", func(t *testing.T) {
			t.Parallel()

			p := newTestPool(t, t.TempDir(), "", &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})
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
			p := newTestPool(t, root, "", &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{})
			require.NoError(t, p.Acquire(context.Background()))

			// heartbeat 更新でエラーにならず、meta が読める状態を維持する。
			require.NoError(t, p.Heartbeat())
			_, ok := p.reg.ReadMeta(1)
			assert.True(t, ok)
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
			reg := NewRegistry(t.TempDir(), "/w/a", "b", 30*time.Minute, 8, func() time.Time { return time.Unix(1000, 0) })
			cfg := Config{Root: "/w/a", SharedProject: "gobp-shared", APIBasePort: 8080, MockAuthBase: 4000}
			p := NewPool(reg, &fakeAdmin{activeBy: map[string]int{}}, &fakeCompose{}, cfg, &out, &out)

			require.NoError(t, p.Status())
			s := out.String()
			assert.Contains(t, s, "SLOT")
			assert.Contains(t, s, "wt1_local")
			assert.Contains(t, s, "free")
		})
	})
}

// 以下は TestPool_Acquire / TestPool_Release で間接的に検証されるため、1:1 規約に従い Skip を明示する。

func TestPool_finishAcquire(t *testing.T) {
	t.Parallel()
	t.Skip("TestPool_Acquire で間接検証")
}

func TestPool_ensureSlotDBs(t *testing.T) {
	t.Parallel()
	t.Skip("TestPool_Acquire（admin.ensured/setup のアサート）で間接検証")
}

func TestPool_writeSlotFile(t *testing.T) {
	t.Parallel()
	t.Skip("TestPool_Acquire（.gobp-db-slot 内容のアサート）で間接検証")
}

func TestPool_printSlotFile(t *testing.T) {
	t.Parallel()
	t.Skip("TestPool_Acquire（stdout への出力）で間接検証")
}
