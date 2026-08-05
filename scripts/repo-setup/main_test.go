package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// commands は、手順を "実行ファイル 引数..." の文字列列へ畳みます。
func commands(steps []step) []string {
	got := make([]string, 0, len(steps))
	for _, s := range steps {
		got = append(got, s.String())
	}

	return got
}

func TestParseLines(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空行と前後の空白を落として行を並べる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"v0.0.0", "v1.0.0"}, parseLines("v0.0.0\n\n  v1.0.0  \n"))
		})

		t.Run("出力が空なら空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, parseLines("\n \n"))
		})
	})
}

func TestTagDeletionSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグごとにローカル削除とリモート削除を並べる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git tag -d v1.0.0",
				"git push origin :refs/tags/v1.0.0",
				"git tag -d v1.1.0",
				"git push origin :refs/tags/v1.1.0",
			}, commands(tagDeletionSteps([]string{"v1.0.0", "v1.1.0"})))
		})

		t.Run("リモート削除だけは失敗を許容する（ローカル限定のタグがあり得るため）", func(t *testing.T) {
			t.Parallel()
			steps := tagDeletionSteps([]string{"v1.0.0"})
			assert.False(t, steps[0].allowFail)
			assert.True(t, steps[1].allowFail)
		})

		t.Run("タグが無ければ手順も空", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, tagDeletionSteps(nil))
		})
	})
}

func TestInitialTagSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注釈付きタグを作ってから push する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git tag -a v0.0.0 -m Initial boilerplate tag",
				"git push origin v0.0.0",
			}, commands(initialTagSteps()))
		})

		t.Run("初期タグの push は失敗を許容しない", func(t *testing.T) {
			t.Parallel()
			for _, s := range initialTagSteps() {
				assert.False(t, s.allowFail)
			}
		})
	})
}

func TestBranchCreationSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存ブランチが無ければ 3 本すべて作る", func(t *testing.T) {
			t.Parallel()
			steps, skipped := branchCreationSteps(nil)
			assert.Equal(t, []string{
				"git branch develop",
				"git branch staging",
				"git branch production",
			}, commands(steps))
			assert.Empty(t, skipped)
		})

		t.Run("既存ブランチは作らずスキップとして返す", func(t *testing.T) {
			t.Parallel()
			steps, skipped := branchCreationSteps([]string{"develop", "production"})
			assert.Equal(t, []string{"git branch staging"}, commands(steps))
			assert.Equal(t, []string{"develop", "production"}, skipped)
		})

		t.Run("すべて既存なら手順は空", func(t *testing.T) {
			t.Parallel()
			steps, skipped := branchCreationSteps([]string{"develop", "staging", "production"})
			assert.Empty(t, steps)
			assert.Len(t, skipped, 3)
		})
	})
}

func TestBranchPushStep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存かどうかに関わらず 3 本まとめて push する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "git push origin develop staging production", branchPushStep().String())
		})
	})
}

func TestDefaultBranchStep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("REST API でデフォルトブランチを production にする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t,
				"gh api -X PATCH repos/example-org/example-api -f default_branch=production",
				defaultBranchStep("example-org/example-api").String())
		})
	})
}

func TestIsReleaseBranch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("release/ を含むブランチは破棄対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isReleaseBranch("release/v1.0.0"))
		})

		t.Run("前方一致ではなく部分一致で判定する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isReleaseBranch("hotfix/release/v1.0.0"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		for _, b := range []string{"develop", "production", "feature/release-notes", ""} {
			t.Run(b+" は破棄対象にしない", func(t *testing.T) {
				t.Parallel()
				assert.False(t, isReleaseBranch(b))
			})
		}
	})
}

func TestOriginalBranchCleanupSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リリースブランチならローカルとリモートの両方を消す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{
				"git branch -D release/v1.0.0",
				"git push origin --delete release/v1.0.0",
			}, commands(originalBranchCleanupSteps("release/v1.0.0")))
		})

		t.Run("リモート削除だけは失敗を許容する（未 push のブランチがあり得るため）", func(t *testing.T) {
			t.Parallel()
			steps := originalBranchCleanupSteps("release/v1.0.0")
			assert.False(t, steps[0].allowFail)
			assert.True(t, steps[1].allowFail)
		})

		t.Run("リリースブランチでなければ何もしない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, originalBranchCleanupSteps("develop"))
		})

		t.Run("ブランチ名が空でも何もしない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, originalBranchCleanupSteps(""))
		})
	})
}

func TestReleaseNotesToDelete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初期タグのノートだけを残す", func(t *testing.T) {
			t.Parallel()
			got := releaseNotesToDelete([]string{"v0.0.0.md", "v1.0.0.md", "v1.1.0.md"})
			assert.Equal(t, []string{"v1.0.0.md", "v1.1.0.md"}, got)
		})

		t.Run("初期タグのノートしか無ければ削除対象は空", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, releaseNotesToDelete([]string{"v0.0.0.md"}))
		})

		t.Run("ディレクトリが空なら削除対象も空", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, releaseNotesToDelete(nil))
		})
	})
}
