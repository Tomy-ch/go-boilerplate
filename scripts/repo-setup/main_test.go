package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// missingCommand は、PATH に存在しない実行ファイル名。失敗する手順を組み立てるために使います。
	missingCommand = "gobp-repo-setup-missing-command"
	// ghRepoViewCall は、リポジトリ名を引く output 呼び出しの記録表記。
	ghRepoViewCall = `gh repo view --json name,owner -q .owner.login + "/" + .name`
	// initialTagCall / initialTagPushCall は、初期タグを打つ 2 手順の記録表記。
	initialTagCall     = "git tag -a v0.0.0 -m Initial boilerplate tag"
	initialTagPushCall = "git push origin v0.0.0"
	// branchPushCall は、用意したブランチをまとめて push する手順の記録表記。
	branchPushCall = "git push origin develop staging production"
)

// errFakeCommand は、差し替えた実行器が返す失敗。
var errFakeCommand = xerrors.New("fake command failed")

// fakeRunner は、git / gh を一切起動せず呼び出しの並びだけを控える実行器。
// bootstrap の手順はタグの一括削除と origin への push を含むため、
// 実リポジトリと GitHub へ触れずに骨格を検証する目的で実行層をここへ差し替えます。
type fakeRunner struct {
	// calls は、run / output が呼ばれた順の「実行ファイル 引数...」表記。
	calls []string
	// outputs は、output が返す標準出力（キーは calls と同じ表記）。
	outputs map[string]string
	// failOn は、この表記の呼び出しだけを失敗させます（空なら常に成功）。
	// allowFail は実行層の責務なのでここでは解釈しません。
	failOn string
	// branches は、branchExists が true を返すブランチ。
	branches map[string]bool
}

// runner は、この実行器を差し込んだ runner を返します。
func (f *fakeRunner) runner() runner {
	return runner{run: f.run, output: f.output, branchExists: f.branchExists}
}

func (f *fakeRunner) run(s step) error {
	call := s.String()
	f.calls = append(f.calls, call)

	if call == f.failOn {
		return errFakeCommand
	}

	return nil
}

func (f *fakeRunner) output(name string, args ...string) (string, error) {
	call := step{name: name, args: args}.String()
	f.calls = append(f.calls, call)

	if call == f.failOn {
		return "", errFakeCommand
	}

	return f.outputs[call], nil
}

func (f *fakeRunner) branchExists(branch string) bool { return f.branches[branch] }

// captureLog は、標準ロガーの出力先をバッファへ差し替え、その内容を読む関数を返します。
// 出力先はプロセス共通のため、使うテストは並列化できません。
func captureLog(t *testing.T) func() string {
	t.Helper()

	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	return buf.String
}

// commands は、手順を "実行ファイル 引数..." の文字列列へ畳みます。
func commands(steps []step) []string {
	got := make([]string, 0, len(steps))
	for _, s := range steps {
		got = append(got, s.String())
	}

	return got
}

// mkdirStep は、指定パスにディレクトリを作る手順を返します。手順が実行されたかを
// ファイルシステム側から観測するために使います。
func mkdirStep(path string) step {
	return step{name: "mkdir", args: []string{path}}
}

// git は、dir のリポジトリに対して git を実行します。作業中のリポジトリへ触れないよう、
// 対象は常に -C で明示します。
func git(t *testing.T, dir string, args ...string) {
	t.Helper()

	//nolint:gosec // 引数はテスト内で組み立てた一時ディレクトリのパスと固定値
	out, err := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	require.NoError(t, err, string(out))
}

// initRepo は、コミットを 1 つ持つ独立した git リポジトリを一時ディレクトリに作り、そのパスを返します。
// リモートは持たせないため、誤って push 系の操作が走っても外へは出ません。
func initRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	git(t, dir, "init", "--quiet")
	git(t, dir, "config", "user.email", "repo-setup-test@example.com")
	git(t, dir, "config", "user.name", "repo-setup-test")
	git(t, dir, "config", "commit.gpgsign", "false")
	git(t, dir, "commit", "--quiet", "--allow-empty", "-m", "initial")

	return dir
}

func Test_step_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行ファイル名と引数を空白でつないだ 1 行にする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "git push origin :refs/tags/v1.0.0",
				step{name: "git", args: []string{"push", "origin", ":refs/tags/v1.0.0"}}.String())
		})

		t.Run("引数が無い場合は実行ファイル名に区切りの空白だけが続く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "gh ", step{name: "gh"}.String())
		})
	})
}

func Test_parseLines(t *testing.T) {
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

func Test_tagDeletionSteps(t *testing.T) {
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

func Test_initialTagSteps(t *testing.T) {
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

		t.Run("タグ作成は失敗を許容しない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, initialTagSteps()[0].allowFail)
		})

		t.Run("初期タグの push は失敗を許容しない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, initialTagSteps()[1].allowFail)
		})
	})
}

func Test_branchCreationSteps(t *testing.T) {
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

func Test_branchPushStep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存かどうかに関わらず 3 本まとめて push する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "git push origin develop staging production", branchPushStep().String())
		})
	})
}

func Test_defaultBranchStep(t *testing.T) {
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

func Test_isReleaseBranch(t *testing.T) {
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

		t.Run("develop は破棄対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isReleaseBranch("develop"))
		})

		t.Run("production は破棄対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isReleaseBranch("production"))
		})

		t.Run("release で始まっても / が続かなければ破棄対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isReleaseBranch("feature/release-notes"))
		})

		t.Run("ブランチ名が空なら破棄対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isReleaseBranch(""))
		})
	})
}

func Test_originalBranchCleanupSteps(t *testing.T) {
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

func Test_releaseNotesToDelete(t *testing.T) {
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

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_tagExists(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("初期タグが存在すれば true", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "tag", initialTag)
			t.Chdir(dir)

			assert.True(t, tagExists(initialTag))
		})

		t.Run("固定のタグ名ではなく引数のタグを見て判定する", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "tag", "v1.2.3")
			t.Chdir(dir)

			assert.True(t, tagExists("v1.2.3"))
			assert.False(t, tagExists(initialTag))
		})

		t.Run("タグが存在しなければ false", func(t *testing.T) {
			t.Chdir(initRepo(t))

			assert.False(t, tagExists(initialTag))
		})

		t.Run("同名のブランチがあってもタグとしては存在しない", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "branch", "v0.0.0")
			t.Chdir(dir)

			assert.False(t, tagExists(initialTag))
		})

		t.Run("git リポジトリでなければ false", func(t *testing.T) {
			t.Chdir(t.TempDir())

			assert.False(t, tagExists(initialTag))
		})
	})
}

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_branchExists(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("ブランチが存在すれば true", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "branch", "develop")
			t.Chdir(dir)

			assert.True(t, branchExists("develop"))
		})

		t.Run("ブランチが存在しなければ false", func(t *testing.T) {
			t.Chdir(initRepo(t))

			assert.False(t, branchExists("develop"))
		})

		t.Run("同名のタグがあってもブランチとしては存在しない", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "tag", "develop")
			t.Chdir(dir)

			assert.False(t, branchExists("develop"))
		})

		t.Run("git リポジトリでなければ false", func(t *testing.T) {
			t.Chdir(t.TempDir())

			assert.False(t, branchExists("develop"))
		})
	})
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コマンドが成功すればエラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(step{name: "true"}))
		})

		t.Run("allowFail の手順は失敗してもエラーにしない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(step{name: missingCommand, allowFail: true}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("allowFail でない手順が失敗したらどの手順かを含むエラーを返す", func(t *testing.T) {
			t.Parallel()
			err := run(step{name: missingCommand, args: []string{"tag", "-d", "v1.0.0"}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed: "+missingCommand+" tag -d v1.0.0")
		})
	})
}

func Test_runner_runAll(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("手順を並び順どおりにすべて実行する", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			first := filepath.Join(dir, "first")
			second := filepath.Join(first, "second")

			require.NoError(t, hostRunner().runAll([]step{mkdirStep(first), mkdirStep(second)}))
			assert.DirExists(t, second)
		})

		t.Run("手順が無ければ何もせず成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, hostRunner().runAll(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗した時点で後続の手順を実行しない", func(t *testing.T) {
			t.Parallel()
			never := filepath.Join(t.TempDir(), "never")

			err := hostRunner().runAll([]step{{name: missingCommand}, mkdirStep(never)})
			require.Error(t, err)
			assert.Contains(t, err.Error(), missingCommand)
			assert.NoDirExists(t, never)
		})
	})
}

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_hostRunner(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("run はホストのコマンドを実行する", func(t *testing.T) {
			require.NoError(t, hostRunner().run(step{name: "true"}))
		})

		t.Run("output はホストのコマンドの標準出力を返す", func(t *testing.T) {
			got, err := hostRunner().output("echo", "example-org/example-api")
			require.NoError(t, err)
			assert.Equal(t, "example-org/example-api\n", got)
		})

		t.Run("branchExists はホストの git でブランチの有無を見る", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "branch", "develop")
			t.Chdir(dir)

			assert.True(t, hostRunner().branchExists("develop"))
			assert.False(t, hostRunner().branchExists("staging"))
		})
	})
}

func Test_output(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("標準出力を改行も落とさずそのまま返す", func(t *testing.T) {
			t.Parallel()
			got, err := output("echo", "example-org/example-api")
			require.NoError(t, err)
			assert.Equal(t, "example-org/example-api\n", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コマンドが失敗したら出力は返さずどの実行かを含むエラーを返す", func(t *testing.T) {
			t.Parallel()
			got, err := output(missingCommand, "repo", "view")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed: "+missingCommand+" repo view")
			assert.Empty(t, got)
		})
	})
}

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_runPreflight(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("初期タグが無ければ初期化を許可する", func(t *testing.T) {
			t.Chdir(initRepo(t))

			require.NoError(t, runPreflight())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("初期タグが既にあれば初期化を止める", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "tag", initialTag)
			t.Chdir(dir)

			err := runPreflight()
			require.ErrorIs(t, err, errInitialTagExists)
		})
	})
}

func Test_runBootstrap(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグの初期化・ブランチ作成・デフォルトブランチの移動をこの順で実行する", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{
				ghRepoViewCall:              "example-org/example-api\n",
				"git branch --show-current": "develop\n",
			}}

			require.NoError(t, runBootstrap(f.runner()))

			tags := slices.Index(f.calls, "git tag")
			branches := slices.Index(f.calls, "git branch develop")
			moved := slices.Index(f.calls, ghRepoViewCall)
			require.NotEqual(t, -1, tags)
			require.NotEqual(t, -1, branches)
			require.NotEqual(t, -1, moved)
			assert.Less(t, tags, branches)
			assert.Less(t, branches, moved)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグの初期化に失敗したらブランチ作成へ進まない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: "git tag"}

			require.ErrorIs(t, runBootstrap(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, "git branch develop")
		})

		t.Run("ブランチ作成に失敗したらデフォルトブランチを移動しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: branchPushCall}

			require.ErrorIs(t, runBootstrap(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, ghRepoViewCall)
		})

		t.Run("デフォルトブランチの移動に失敗したらエラーを返す", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{ghRepoViewCall: "example-org/example-api\n"},
				failOn:  "gh api -X PATCH repos/example-org/example-api -f default_branch=production",
			}

			require.ErrorIs(t, runBootstrap(f.runner()), errFakeCommand)
		})
	})
}

func Test_resetTags(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存タグをローカルとリモートから消してから初期タグを打つ", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{"git tag": "v1.0.0\nv1.1.0\n"}}

			require.NoError(t, resetTags(f.runner()))
			assert.Equal(t, []string{
				"git tag",
				"git tag -d v1.0.0",
				"git push origin :refs/tags/v1.0.0",
				"git tag -d v1.1.0",
				"git push origin :refs/tags/v1.1.0",
				initialTagCall,
				initialTagPushCall,
			}, f.calls)
		})

		t.Run("タグが 1 件も無ければ削除手順を挟まず初期タグだけを打つ", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{"git tag": "\n  \n"}}

			require.NoError(t, resetTags(f.runner()))
			assert.Equal(t, []string{"git tag", initialTagCall, initialTagPushCall}, f.calls)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグ一覧を取得できなければ 1 つも消さない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: "git tag"}

			require.ErrorIs(t, resetTags(f.runner()), errFakeCommand)
			assert.Equal(t, []string{"git tag"}, f.calls)
		})

		t.Run("タグの削除に失敗したら初期タグを打たない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{"git tag": "v1.0.0\nv1.1.0\n"},
				failOn:  "git tag -d v1.0.0",
			}

			require.ErrorIs(t, resetTags(f.runner()), errFakeCommand)
			assert.Equal(t, []string{"git tag", "git tag -d v1.0.0"}, f.calls)
		})

		t.Run("初期タグの作成に失敗したら push しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: initialTagCall}

			require.ErrorIs(t, resetTags(f.runner()), errFakeCommand)
			assert.Equal(t, []string{"git tag", initialTagCall}, f.calls)
		})
	})
}

//nolint:paralleltest // ログの出力先はプロセス共通のため並列化できない
func Test_createBranches(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("既存ブランチが無ければ 3 本作ってからまとめて push する", func(t *testing.T) {
			f := &fakeRunner{}

			require.NoError(t, createBranches(f.runner()))
			assert.Equal(t, []string{
				"git branch develop",
				"git branch staging",
				"git branch production",
				branchPushCall,
			}, f.calls)
		})

		t.Run("既に在るブランチは作らずスキップしたことを知らせる", func(t *testing.T) {
			logged := captureLog(t)
			f := &fakeRunner{branches: map[string]bool{"develop": true}}

			require.NoError(t, createBranches(f.runner()))
			assert.Equal(t, []string{"git branch staging", "git branch production", branchPushCall}, f.calls)
			assert.Contains(t, logged(), "ブランチ 【develop】 は既に存在します")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("ブランチの作成に失敗したら push しない", func(t *testing.T) {
			f := &fakeRunner{failOn: "git branch staging"}

			require.ErrorIs(t, createBranches(f.runner()), errFakeCommand)
			assert.Equal(t, []string{"git branch develop", "git branch staging"}, f.calls)
		})

		t.Run("push に失敗したらエラーを返す", func(t *testing.T) {
			f := &fakeRunner{failOn: branchPushCall}

			require.ErrorIs(t, createBranches(f.runner()), errFakeCommand)
		})
	})
}

func Test_moveDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリ名を引いてデフォルトブランチを移し、元のリリースブランチを片付ける", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{
				ghRepoViewCall:              "example-org/example-api\n",
				"git branch --show-current": "release/v1.0.0\n",
			}}

			require.NoError(t, moveDefaultBranch(f.runner()))
			assert.Equal(t, []string{
				ghRepoViewCall,
				"gh api -X PATCH repos/example-org/example-api -f default_branch=production",
				"git fetch --prune",
				"git branch --show-current",
				"git switch production",
				"git branch -D release/v1.0.0",
				"git push origin --delete release/v1.0.0",
			}, f.calls)
		})

		t.Run("元のブランチがリリースブランチでなければ何も消さない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{
				ghRepoViewCall:              "example-org/example-api\n",
				"git branch --show-current": "develop\n",
			}}

			require.NoError(t, moveDefaultBranch(f.runner()))
			assert.Equal(t, "git switch production", f.calls[len(f.calls)-1])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリ名を取得できなければデフォルトブランチを変更しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: ghRepoViewCall}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
			assert.Equal(t, []string{ghRepoViewCall}, f.calls)
		})

		t.Run("デフォルトブランチの変更に失敗したら switch しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{ghRepoViewCall: "example-org/example-api\n"},
				failOn:  "gh api -X PATCH repos/example-org/example-api -f default_branch=production",
			}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, "git switch production")
		})

		t.Run("fetch に失敗したら switch しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{ghRepoViewCall: "example-org/example-api\n"},
				failOn:  "git fetch --prune",
			}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, "git switch production")
		})

		t.Run("元のブランチ名を取得できなければ switch しない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{ghRepoViewCall: "example-org/example-api\n"},
				failOn:  "git branch --show-current",
			}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, "git switch production")
		})

		t.Run("デフォルトブランチへ移れなければ元のブランチを消さない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{
					ghRepoViewCall:              "example-org/example-api\n",
					"git branch --show-current": "release/v1.0.0\n",
				},
				failOn: "git switch production",
			}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
			assert.NotContains(t, f.calls, "git branch -D release/v1.0.0")
			assert.NotContains(t, f.calls, "git push origin --delete release/v1.0.0")
		})

		t.Run("元のブランチの片付けに失敗したら成功として終えない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{
				outputs: map[string]string{
					ghRepoViewCall:              "example-org/example-api\n",
					"git branch --show-current": "release/v1.0.0\n",
				},
				failOn: "git branch -D release/v1.0.0",
			}

			require.ErrorIs(t, moveDefaultBranch(f.runner()), errFakeCommand)
		})
	})
}

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_runPruneReleaseNotes(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("初期タグのノートとディレクトリを残して他のノートを消す", func(t *testing.T) {
			dir := t.TempDir()
			noteDir := filepath.Join(dir, releaseNoteDir)
			require.NoError(t, os.MkdirAll(filepath.Join(noteDir, "archive"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(noteDir, initialTag+".md"), []byte("keep"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(noteDir, "v1.0.0.md"), []byte("drop"), 0o600))
			t.Chdir(dir)

			require.NoError(t, runPruneReleaseNotes())
			assert.FileExists(t, filepath.Join(noteDir, initialTag+".md"))
			assert.NoFileExists(t, filepath.Join(noteDir, "v1.0.0.md"))
			assert.DirExists(t, filepath.Join(noteDir, "archive"))
		})

		t.Run("リリースノートのディレクトリが無ければ何もせず成功する", func(t *testing.T) {
			t.Chdir(t.TempDir())

			require.NoError(t, runPruneReleaseNotes())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("置き場所がディレクトリとして読めなければ黙って飛ばさずエラーにする", func(t *testing.T) {
			dir := t.TempDir()
			notePath := filepath.Join(dir, releaseNoteDir)
			require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o750))
			require.NoError(t, os.WriteFile(notePath, nil, 0o600))
			t.Chdir(dir)

			require.Error(t, runPruneReleaseNotes())
		})

		// 消せなかったノートを黙って飛ばすと、初期化後のリポジトリに前身のリリース履歴が残る。
		t.Run("ノートを消せなければどのファイルかを含むエラーにする", func(t *testing.T) {
			if os.Geteuid() == 0 {
				t.Skip("特権実行では書き込み権限を落としても unlink が通ってしまい、削除の失敗を作れない")
			}

			dir := t.TempDir()
			noteDir := filepath.Join(dir, releaseNoteDir)
			require.NoError(t, os.MkdirAll(noteDir, 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(noteDir, "v1.0.0.md"), []byte("drop"), 0o600))
			// ディレクトリは走査に実行ビットが要るため 0600 以下にはできない。書き込みビットだけを落とす。
			require.NoError(t, os.Chmod(noteDir, 0o500))       //nolint:gosec // 削除を失敗させるためディレクトリの権限を落とす
			t.Cleanup(func() { _ = os.Chmod(noteDir, 0o750) }) //nolint:gosec // t.TempDir を後片付けできるよう権限を戻す
			t.Chdir(dir)

			err := runPruneReleaseNotes()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to remove v1.0.0.md")
		})
	})
}

// bootstrappableRunner は、bootstrap の手順を最後まで通せる出力を備えた実行器を返します。
func bootstrappableRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{
		ghRepoViewCall:              "example-org/example-api\n",
		"git branch --show-current": "develop\n",
	}}
}

// writeReleaseNotes は、初期タグのノートと破棄対象のノートを持つ作業ディレクトリへ移り、
// ノートの置き場所の絶対パスを返します。
func writeReleaseNotes(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	noteDir := filepath.Join(dir, releaseNoteDir)
	require.NoError(t, os.MkdirAll(noteDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(noteDir, initialTag+".md"), []byte("keep"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(noteDir, "v1.0.0.md"), []byte("drop"), 0o600))
	t.Chdir(dir)

	return noteDir
}

//nolint:paralleltest // t.Chdir はプロセス全体の作業ディレクトリを変えるため並列化できない
func Test_execute(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("preflight は実行層を使わず初期タグの有無だけを見る", func(t *testing.T) {
			t.Chdir(initRepo(t))
			f := bootstrappableRunner()

			require.NoError(t, execute(f.runner(), []string{"preflight"}))
			assert.Empty(t, f.calls)
		})

		t.Run("bootstrap は差し替えた実行層で手順を進める", func(t *testing.T) {
			f := bootstrappableRunner()

			require.NoError(t, execute(f.runner(), []string{"bootstrap"}))
			assert.Contains(t, f.calls, initialTagCall)
			assert.Contains(t, f.calls, branchPushCall)
		})

		t.Run("prune-release-notes は初期タグ以外のノートを消す", func(t *testing.T) {
			noteDir := writeReleaseNotes(t)
			f := bootstrappableRunner()

			require.NoError(t, execute(f.runner(), []string{"prune-release-notes"}))
			assert.FileExists(t, filepath.Join(noteDir, initialTag+".md"))
			assert.NoFileExists(t, filepath.Join(noteDir, "v1.0.0.md"))
			assert.Empty(t, f.calls)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("サブコマンドが無ければ使い方を示して手順を 1 つも実行しない", func(t *testing.T) {
			f := bootstrappableRunner()

			require.ErrorIs(t, execute(f.runner(), nil), errUsage)
			assert.Empty(t, f.calls)
		})

		t.Run("未知のサブコマンドでは手順を 1 つも実行しない", func(t *testing.T) {
			f := bootstrappableRunner()

			require.ErrorIs(t, execute(f.runner(), []string{"bogus"}), errUnknownSubcommand)
			assert.Empty(t, f.calls)
		})

		t.Run("preflight の中止をそのまま返す", func(t *testing.T) {
			dir := initRepo(t)
			git(t, dir, "tag", initialTag)
			t.Chdir(dir)

			require.ErrorIs(t, execute(bootstrappableRunner().runner(), []string{"preflight"}), errInitialTagExists)
		})

		t.Run("bootstrap の失敗をそのまま返す", func(t *testing.T) {
			f := &fakeRunner{failOn: "git tag"}

			require.ErrorIs(t, execute(f.runner(), []string{"bootstrap"}), errFakeCommand)
		})

		t.Run("prune-release-notes の失敗をそのまま返す", func(t *testing.T) {
			dir := t.TempDir()
			notePath := filepath.Join(dir, releaseNoteDir)
			require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o750))
			require.NoError(t, os.WriteFile(notePath, nil, 0o600))
			t.Chdir(dir)

			require.Error(t, execute(bootstrappableRunner().runner(), []string{"prune-release-notes"}))
		})
	})
}
