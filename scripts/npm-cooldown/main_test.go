package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockJSON は lockfileVersion 3 の最小構造を組み立てる。
const lockJSON = `{
  "lockfileVersion": 3,
  "packages": {
    "": { "name": "root", "version": "1.0.0" },
    "node_modules/brace-expansion": { "version": "5.0.7" },
    "node_modules/@stoplight/spectral-core/node_modules/brace-expansion": { "version": "1.1.16" },
    "node_modules/@scope/pkg": { "version": "2.3.4" },
    "node_modules/linked": { "version": "1.0.0", "link": true },
    "node_modules/no-version": {},
    "workspaces/app": { "version": "0.1.0" }
  }
}`

func writeNpmrc(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".npmrc"), []byte(body), 0o600))
}

func TestParseLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("node_modules 配下の実パッケージだけを取り出す", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			keys := make([]string, 0, len(got))
			for _, e := range got {
				keys = append(keys, e.key())
			}
			assert.ElementsMatch(t, []string{
				"brace-expansion@5.0.7",
				"brace-expansion@1.1.16",
				"@scope/pkg@2.3.4",
			}, keys)
		})

		t.Run("ネストしたコピーは最後の node_modules 以降を名前とする", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			var nested entry
			for _, e := range got {
				if e.version == "1.1.16" {
					nested = e
				}
			}
			assert.Equal(t, "brace-expansion", nested.name)
			assert.Equal(t, "node_modules/@stoplight/spectral-core/node_modules/brace-expansion", nested.path)
		})

		t.Run("スコープ付きパッケージ名を保持する", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			var scoped entry
			for _, e := range got {
				if e.version == "2.3.4" {
					scoped = e
				}
			}
			assert.Equal(t, "@scope/pkg", scoped.name)
		})

		t.Run("ルート・link・version 無し・workspace は対象外", func(t *testing.T) {
			t.Parallel()
			got, err := parseLock([]byte(lockJSON))
			require.NoError(t, err)

			for _, e := range got {
				assert.NotEqual(t, "linked", e.name)
				assert.NotEqual(t, "no-version", e.name)
				assert.NotEqual(t, "workspaces/app", e.path)
			}
			assert.Len(t, got, 3)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON として壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := parseLock([]byte("{"))
			require.Error(t, err)
		})
	})
}

func TestMinReleaseAge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("min-release-age の宣言値を読む", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=7\n")

			got, err := minReleaseAge(root, "package-lock.json")
			require.NoError(t, err)
			assert.Equal(t, 7, got)
		})

		t.Run("コメント行と前後の空白を無視する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "# min-release-age=99\n; min-release-age=98\n  min-release-age = 14 \n")

			got, err := minReleaseAge(root, "package-lock.json")
			require.NoError(t, err)
			assert.Equal(t, 14, got)
		})

		t.Run("lockfile と同じ階層の .npmrc を見る", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			dir := filepath.Join(root, "docker", "tools")
			require.NoError(t, os.MkdirAll(dir, 0o750))
			writeNpmrc(t, dir, "min-release-age=3\n")

			got, err := minReleaseAge(root, filepath.Join("docker", "tools", "package-lock.json"))
			require.NoError(t, err)
			assert.Equal(t, 3, got)
		})

		t.Run(".npmrc が無ければ 0（守るべき cooldown なし）", func(t *testing.T) {
			t.Parallel()
			got, err := minReleaseAge(t.TempDir(), "package-lock.json")
			require.NoError(t, err)
			assert.Zero(t, got)
		})

		t.Run("min-release-age の宣言が無ければ 0", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "registry=https://example.test/\n")

			got, err := minReleaseAge(root, "package-lock.json")
			require.NoError(t, err)
			assert.Zero(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数値として解釈できない値はエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeNpmrc(t, root, "min-release-age=seven\n")

			_, err := minReleaseAge(root, "package-lock.json")
			require.Error(t, err)
		})
	})
}

func TestLockfiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("node_modules と vendor 配下を除外して収集する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, rel := range []string{
				"docker/tools",
				"docker/mock-auth-server",
				"docker/tools/node_modules/dep",
				"vendor/somepkg",
			} {
				require.NoError(t, os.MkdirAll(filepath.Join(root, rel), 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(root, rel, "package-lock.json"), []byte("{}"), 0o600))
			}

			got, err := lockfiles(root)
			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join("docker", "mock-auth-server", "package-lock.json"),
				filepath.Join("docker", "tools", "package-lock.json"),
			}, got)
		})

		t.Run("lockfile が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			got, err := lockfiles(t.TempDir())
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})
}

func TestEntryKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name@version を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "@scope/pkg@2.3.4", entry{name: "@scope/pkg", version: "2.3.4"}.key())
		})
	})
}

func TestSummary(t *testing.T) {
	t.Parallel()

	fs := []finding{{
		lockfile:  "docker/tools/package-lock.json",
		entry:     entry{name: "brace-expansion", version: "5.0.8"},
		published: time.Date(2026, time.July, 23, 11, 39, 25, 0, time.UTC),
		ageDays:   2,
		minAge:    7,
	}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ監査件数のみを述べる", func(t *testing.T) {
			t.Parallel()
			got := summary(nil, 504, "")
			assert.Contains(t, got, "504")
			assert.NotContains(t, got, "min-release-age を満たしていません")
		})

		t.Run("違反はパッケージ・版・経過日数・lockfile を列挙する", func(t *testing.T) {
			t.Parallel()
			got := summary(fs, 12, "origin/release/v2.2.0")

			assert.Contains(t, got, "brace-expansion")
			assert.Contains(t, got, "5.0.8")
			assert.Contains(t, got, "公開 2 日 (< 7)")
			assert.Contains(t, got, "docker/tools/package-lock.json")
		})

		t.Run("バイパスが意図的なら根拠を残すよう促す", func(t *testing.T) {
			t.Parallel()
			got := summary(fs, 12, "")
			assert.Contains(t, got, "判断根拠")
		})

		t.Run("base 指定の有無でスコープの表現が変わる", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, summary(nil, 1, ""), "全エントリ")
			assert.Contains(t, summary(nil, 1, "origin/main"), "追加/変更された")
		})
	})
}

func TestAppendOutput(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("findings と audited を追記する", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out")
			require.NoError(t, appendOutput(path, 3, 504))

			b, err := os.ReadFile(path) //nolint:gosec // テスト内の一時ファイル
			require.NoError(t, err)
			assert.Contains(t, string(b), "findings=3")
			assert.Contains(t, string(b), "audited=504")
		})
	})
}
