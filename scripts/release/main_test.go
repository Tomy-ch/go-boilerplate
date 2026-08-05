package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commands は、手順を "実行ファイル 引数..." の文字列列へ畳みます。
func commands(steps []step) []string {
	got := make([]string, 0, len(steps))
	for _, s := range steps {
		got = append(got, s.String())
	}

	return got
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("vX.Y.Z を各要素へ分解する", func(t *testing.T) {
			t.Parallel()
			v, ok := parseVersion("v1.2.3")
			require.True(t, ok)
			assert.Equal(t, version{major: 1, minor: 2, patch: 3}, v)
		})

		t.Run("前後の空白を無視する", func(t *testing.T) {
			t.Parallel()
			v, ok := parseVersion("  v0.0.0\r")
			require.True(t, ok)
			assert.Equal(t, "v0.0.0", v.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		for _, s := range []string{"1.2.3", "v1.2", "v1.2.3.4", "v1.2.3-rc1", "release/v1.2.3", ""} {
			t.Run("リリースタグ形式でない "+s+" は受け付けない", func(t *testing.T) {
				t.Parallel()
				_, ok := parseVersion(s)
				assert.False(t, ok)
			})
		}
	})
}

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数値順で比較するため v1.10.0 は v1.9.0 より新しい", func(t *testing.T) {
			t.Parallel()
			v, ok := latestVersion("v1.9.0\nv1.10.0\nv1.2.0\n")
			require.True(t, ok)
			assert.Equal(t, "v1.10.0", v.String())
		})

		t.Run("リリースタグ以外の行は無視する", func(t *testing.T) {
			t.Parallel()
			v, ok := latestVersion("v0.1.0\nnightly\nv1.0.0-rc1\nv0.2.0\n")
			require.True(t, ok)
			assert.Equal(t, "v0.2.0", v.String())
		})

		t.Run("major が大きいものを優先する", func(t *testing.T) {
			t.Parallel()
			v, ok := latestVersion("v2.0.0\nv1.99.99\n")
			require.True(t, ok)
			assert.Equal(t, "v2.0.0", v.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リリースタグが 1 つも無ければ見つからないと返す", func(t *testing.T) {
			t.Parallel()
			_, ok := latestVersion("nightly\nv1.0.0-rc1\n")
			assert.False(t, ok)
		})

		t.Run("出力が空でも見つからないと返す", func(t *testing.T) {
			t.Parallel()
			_, ok := latestVersion("")
			assert.False(t, ok)
		})
	})
}

func TestBump(t *testing.T) {
	t.Parallel()

	base := version{major: 1, minor: 2, patch: 3}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("patch は最下位だけを進める", func(t *testing.T) {
			t.Parallel()
			v, err := bump(base, "patch")
			require.NoError(t, err)
			assert.Equal(t, "v1.2.4", v.String())
		})

		t.Run("minor は patch を 0 へ戻す", func(t *testing.T) {
			t.Parallel()
			v, err := bump(base, "minor")
			require.NoError(t, err)
			assert.Equal(t, "v1.3.0", v.String())
		})

		t.Run("major は minor と patch を 0 へ戻す", func(t *testing.T) {
			t.Parallel()
			v, err := bump(base, "major")
			require.NoError(t, err)
			assert.Equal(t, "v2.0.0", v.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の粒度は拒否する", func(t *testing.T) {
			t.Parallel()
			_, err := bump(base, "hotfix")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown -bump")
		})
	})
}

func TestSyncProductionSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("production を origin の最新へ揃えてからタグを取り直す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git fetch origin production",
				"git switch production",
				"git reset --hard origin/production",
				"git fetch --tags origin",
			}, commands(syncProductionSteps()))
		})
	})
}

func TestTagSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグ作成 → push → GitHub Release の順に並べる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git tag -a v1.2.4 -F .github/release/v1.2.4.md",
				"git push origin v1.2.4",
				"gh release create v1.2.4 --title v1.2.4 --notes-file .github/release/v1.2.4.md",
			}, commands(tagSteps("v1.2.4", notePath("v1.2.4"))))
		})

		t.Run("タグ本文と Release 本文は同じリリースノートを指す", func(t *testing.T) {
			t.Parallel()
			steps := tagSteps("v9.9.9", notePath("v9.9.9"))
			assert.Equal(t, ".github/release/v9.9.9.md", steps[0].args[len(steps[0].args)-1])
			assert.Equal(t, ".github/release/v9.9.9.md", steps[2].args[len(steps[2].args)-1])
		})
	})
}

func TestBranchSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base から切って push した後にデフォルトブランチを変える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git fetch origin production",
				"git switch -c release/v1.3.0 origin/production",
				"git push origin release/v1.3.0",
				"gh repo edit --default-branch release/v1.3.0",
			}, commands(branchSteps("production", "release/v1.3.0")))
		})

		t.Run("デフォルトブランチの切り替えは push の後に置く", func(t *testing.T) {
			t.Parallel()
			steps := branchSteps("production", "hotfix/v1.2.4")
			assert.Equal(t, "git", steps[2].name)
			assert.Equal(t, "gh", steps[3].name)
		})
	})
}

func TestNotePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リリースノートは .github/release 配下の <version>.md", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, ".github/release/v1.0.0.md", notePath("v1.0.0"))
		})
	})
}
