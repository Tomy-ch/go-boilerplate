package main

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// tagListCall は、タグ一覧を引く output 呼び出しの記録表記。
	tagListCall = "git tag"
	// fetchTagsCall は、タグを取り直す手順の記録表記。
	fetchTagsCall = "git fetch --tags origin"
	// annotateTagCall / tagPushCall / releaseCreateCall は、取り消しの効かない 3 手順の記録表記。
	annotateTagCall   = "git tag -a v1.2.4 -F .github/release/v1.2.4.md"
	tagPushCall       = "git push origin v1.2.4"
	releaseCreateCall = "gh release create v1.2.4 --title v1.2.4 --notes-file .github/release/v1.2.4.md"
	// branchCreateCall / branchPushCall / defaultBranchCall は、ブランチ作成側の後戻りできない手順の記録表記。
	branchCreateCall  = "git switch -c release/v1.3.0 origin/production"
	branchPushCall    = "git push origin release/v1.3.0"
	defaultBranchCall = "gh repo edit --default-branch release/v1.3.0"
)

// errFakeCommand は、差し替えた実行器が返す失敗。
var errFakeCommand = xerrors.New("fake command failed")

// fakeRunner は、git / gh を一切起動せず呼び出しの並びだけを控える実行器。
// tag / branch の手順はタグの push と GitHub Release の作成を含み、走らせてしまうと
// 取り消せないため、中止条件と手順の順序はここへ差し替えた実行層の上で検証します。
type fakeRunner struct {
	// calls は、run / output が呼ばれた順の「実行ファイル 引数...」表記。
	calls []string
	// outputs は、output が返す標準出力（キーは calls と同じ表記）。
	outputs map[string]string
	// failOn は、この表記の呼び出しだけを失敗させます（空なら常に成功）。
	failOn string
	// remoteBranches は、remoteBranchExists が true を返すブランチ。
	remoteBranches map[string]bool
}

// runner は、この実行器を差し込んだ runner を返します。
func (f *fakeRunner) runner() runner {
	return runner{run: f.run, output: f.output, remoteBranchExists: f.remoteBranchExists}
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

func (f *fakeRunner) remoteBranchExists(branch string) bool { return f.remoteBranches[branch] }

// taggableRunner は、v1.2.3 を最新タグとして返す実行器を用意します。
func taggableRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{tagListCall: "v1.2.3\n"}}
}

// writeNote は、作業ディレクトリを一時ディレクトリへ移し、v1.2.3 の次のパッチ版にあたる
// リリースノートを置きます。
func writeNote(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, releaseNoteDir), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, notePath("v1.2.4")), []byte("note\n"), 0o600))
	t.Chdir(dir)
}

// commands は、手順を "実行ファイル 引数..." の文字列列へ畳みます。
func commands(steps []step) []string {
	got := make([]string, 0, len(steps))
	for _, s := range steps {
		got = append(got, s.String())
	}

	return got
}

// gitIn は、一時リポジトリに対して git を実行します。ホストの設定に左右されないよう
// 著者と署名を固定します。
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()

	base := []string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}

	//nolint:gosec // 引数は本ファイル内のリテラルと t.TempDir のパス
	out, err := exec.CommandContext(t.Context(), "git", append(base, args...)...).CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// newRepo は、origin を自分自身へ向けた production ブランチ 1 本のリポジトリを用意します。
// origin がローカルパスなので、fetch も ls-remote も実際の GitHub には触れません。
func newRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "production")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o600))
	gitIn(t, dir, "add", "README.md")
	gitIn(t, dir, "commit", "-q", "-m", "init")
	gitIn(t, dir, "remote", "add", "origin", dir)
	gitIn(t, dir, "fetch", "-q", "--tags", "origin")

	return dir
}

func Test_parseVersion(t *testing.T) {
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

		t.Run("接頭辞 v を欠く 1.2.3 は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("1.2.3")
			assert.False(t, ok)
		})

		t.Run("patch を欠く v1.2 は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("v1.2")
			assert.False(t, ok)
		})

		t.Run("要素が多い v1.2.3.4 は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("v1.2.3.4")
			assert.False(t, ok)
		})

		t.Run("プレリリース付きの v1.2.3-rc1 は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("v1.2.3-rc1")
			assert.False(t, ok)
		})

		t.Run("リリースブランチ名 release/v1.2.3 は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("release/v1.2.3")
			assert.False(t, ok)
		})

		t.Run("空文字は受け付けない", func(t *testing.T) {
			t.Parallel()
			_, ok := parseVersion("")
			assert.False(t, ok)
		})
	})
}

func Test_latestVersion(t *testing.T) {
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

		// major と minor が並んだ場合の順序が決まらないと、起点を取り違えて既存タグへ
		// 打ち直すか、番号を飛ばした次バージョンになる。
		t.Run("major と minor が同じなら patch が大きいものを優先する", func(t *testing.T) {
			t.Parallel()
			v, ok := latestVersion("v1.2.3\nv1.2.10\nv1.2.4\n")
			require.True(t, ok)
			assert.Equal(t, "v1.2.10", v.String())
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

func Test_bump(t *testing.T) {
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
			require.ErrorIs(t, err, errUnknownBump)
			assert.Contains(t, err.Error(), "unknown -bump")
		})
	})
}

func Test_syncProductionSteps(t *testing.T) {
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

func Test_tagSteps(t *testing.T) {
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

func Test_branchSteps(t *testing.T) {
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

func Test_notePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リリースノートは .github/release 配下の <version>.md", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, ".github/release/v1.0.0.md", notePath("v1.0.0"))
		})
	})
}

func Test_version_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各要素をドットで繋いだ vX.Y.Z を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "v1.10.2", version{major: 1, minor: 10, patch: 2}.String())
		})

		// タグ名は Release とリリースノートのパスの両方に使われるため、ゼロ値も省略しない。
		t.Run("ゼロの要素も省略しない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "v0.0.0", version{}.String())
		})
	})
}

func Test_step_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行ファイル名と引数を空白で繋ぐ", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "git push origin v1.0.0", step{name: "git", args: []string{"push", "origin", "v1.0.0"}}.String())
		})

		t.Run("引数を持たない手順でも実行ファイル名が分かる", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, step{name: "gh"}.String(), "gh")
		})
	})
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功したコマンドはエラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(step{name: "git", args: []string{"--version"}}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// どの手順で落ちたかが分からないと、途中まで進んだ状態から手作業で復旧できない。
		t.Run("失敗したコマンドはどの手順かを含めて返す", func(t *testing.T) {
			t.Parallel()
			err := run(step{name: "git", args: []string{"no-such-subcommand"}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed: git no-such-subcommand")
		})
	})
}

func Test_runner_runAll(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("与えた手順を順に実行する", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
			require.NoError(t, hostRunner().runAll([]step{
				{name: "touch", args: []string{first}},
				{name: "touch", args: []string{second}},
			}))
			assert.FileExists(t, first)
			assert.FileExists(t, second)
		})

		t.Run("手順が無ければ何もせず成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, hostRunner().runAll(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 手順にはタグの push や Release の作成が並ぶため、失敗を跨いで進むと取り消せない。
		t.Run("失敗した時点で以降の手順を実行しない", func(t *testing.T) {
			t.Parallel()
			skipped := filepath.Join(t.TempDir(), "skipped")
			err := hostRunner().runAll([]step{
				{name: "git", args: []string{"no-such-subcommand"}},
				{name: "touch", args: []string{skipped}},
			})
			require.Error(t, err)
			assert.NoFileExists(t, skipped)
		})
	})
}

//nolint:paralleltest // t.Chdir でプロセスの作業ディレクトリを変えるため並列化不可
func Test_hostRunner(t *testing.T) {
	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("run はホストのコマンドを実行する", func(t *testing.T) {
			require.NoError(t, hostRunner().run(step{name: "git", args: []string{"--version"}}))
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("output はホストのコマンドの標準出力を返す", func(t *testing.T) {
			got, err := hostRunner().output("echo", "v1.0.0")
			require.NoError(t, err)
			assert.Equal(t, "v1.0.0\n", got)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("remoteBranchExists はホストの git で origin のブランチを見る", func(t *testing.T) {
			t.Chdir(newRepo(t))

			assert.True(t, hostRunner().remoteBranchExists("production"))
			assert.False(t, hostRunner().remoteBranchExists("release/v9.9.9"))
		})
	})
}

func Test_output(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// 出力はタグ一覧と作業ツリーの汚れ判定の入力になるため、加工せずに渡す。
		t.Run("標準出力をそのまま返す", func(t *testing.T) {
			t.Parallel()
			got, err := output("echo", "hello")
			require.NoError(t, err)
			assert.Equal(t, "hello\n", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗したコマンドは引数まで含めて返す", func(t *testing.T) {
			t.Parallel()
			got, err := output("git", "no-such-subcommand")
			require.Error(t, err)
			assert.Empty(t, got)
			assert.Contains(t, err.Error(), "failed: git no-such-subcommand")
		})
	})
}

//nolint:paralleltest // t.Chdir でプロセスの作業ディレクトリを変えるため並列化不可
func Test_remoteBranchExists(t *testing.T) {
	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		// 既存ブランチを見落とすと、リリースブランチを取り違えたまま push まで進む。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("origin に同名ブランチがあれば真を返す", func(t *testing.T) {
			dir := newRepo(t)
			t.Chdir(dir)
			assert.True(t, remoteBranchExists("production"))
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("origin に無いブランチには偽を返す", func(t *testing.T) {
			dir := newRepo(t)
			t.Chdir(dir)
			assert.False(t, remoteBranchExists("release/v9.9.9"))
		})
	})
}

func Test_resolveNext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// origin のタグを取り直さずに判定すると、他所で打たれた最新タグを見落として
		// 既存のバージョンへ二重にタグを打つ。
		t.Run("タグを取り直してから最新のタグを起点に次バージョンを決める", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{outputs: map[string]string{tagListCall: "v1.2.3\nv1.10.0\n"}}

			current, next, err := resolveNext(f.runner(), "minor")
			require.NoError(t, err)
			assert.Equal(t, "v1.10.0", current.String())
			assert.Equal(t, "v1.11.0", next.String())
			assert.Equal(t, []string{fetchTagsCall, tagListCall}, f.calls)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タグを取り直せなければタグ一覧を見ない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: fetchTagsCall}

			_, _, err := resolveNext(f.runner(), "patch")
			require.ErrorIs(t, err, errFakeCommand)
			assert.Equal(t, []string{fetchTagsCall}, f.calls)
		})

		t.Run("タグ一覧を取得できなければエラーにする", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: tagListCall}

			_, _, err := resolveNext(f.runner(), "patch")
			require.ErrorIs(t, err, errFakeCommand)
		})

		t.Run("起点となるタグが無ければ初期タグを促す", func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveNext((&fakeRunner{}).runner(), "patch")
			require.ErrorIs(t, err, errNoTag)
		})

		t.Run("未知の粒度は次バージョンを決めずに返す", func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveNext(taggableRunner().runner(), "hotfix")
			require.ErrorIs(t, err, errUnknownBump)
		})
	})
}

//nolint:paralleltest // t.Chdir でプロセスの作業ディレクトリを変えるため並列化不可
func Test_runTag(t *testing.T) {
	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		// タグは production HEAD に打つ。production へ合わせる前にノートを見ると、
		// 切り替え前の作業ツリーにあるノートで判定してしまう。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("production へ合わせ、ノートを確かめてからタグ・push・Release の順に進む", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()

			require.NoError(t, runTag(f.runner(), []string{"-bump", "patch"}))
			assert.Equal(t, []string{
				fetchTagsCall,
				tagListCall,
				"git fetch origin production",
				"git switch production",
				"git reset --hard origin/production",
				fetchTagsCall,
				annotateTagCall,
				tagPushCall,
				releaseCreateCall,
			}, f.calls)
		})
	})

	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("解釈できないフラグでは手順を 1 つも実行しない", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()

			err := runTag(f.runner(), []string{"-no-such-flag"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse flags")
			assert.Empty(t, f.calls)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("未知の粒度では production へ切り替える前に中止する", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()

			require.ErrorIs(t, runTag(f.runner(), []string{"-bump", "bogus"}), errUnknownBump)
			assert.NotContains(t, f.calls, "git switch production")
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("production へ合わせられなければノートを見ずにタグを打たない", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()
			f.failOn = "git reset --hard origin/production"

			require.ErrorIs(t, runTag(f.runner(), []string{"-bump", "patch"}), errFakeCommand)
			assert.NotContains(t, f.calls, annotateTagCall)
		})

		// ノートはタグと Release の本文になるので、無いまま進むと本文の無いリリースが残る。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("production にノートが無ければ production へ合わせた後で中止する", func(t *testing.T) {
			t.Chdir(t.TempDir())
			f := taggableRunner()

			require.ErrorIs(t, runTag(f.runner(), []string{"-bump", "patch"}), errNoReleaseNote)
			assert.Contains(t, f.calls, "git switch production")
			assert.NotContains(t, f.calls, annotateTagCall)
			assert.NotContains(t, f.calls, tagPushCall)
			assert.NotContains(t, f.calls, releaseCreateCall)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("タグを作れなければ push も Release 作成も行わない", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()
			f.failOn = annotateTagCall

			require.ErrorIs(t, runTag(f.runner(), []string{"-bump", "patch"}), errFakeCommand)
			assert.NotContains(t, f.calls, tagPushCall)
			assert.NotContains(t, f.calls, releaseCreateCall)
		})

		// push できていないタグで Release を作ると、参照先の無いリリースが GitHub に残る。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("タグを push できなければ Release を作らない", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()
			f.failOn = tagPushCall

			require.ErrorIs(t, runTag(f.runner(), []string{"-bump", "patch"}), errFakeCommand)
			assert.NotContains(t, f.calls, releaseCreateCall)
		})
	})
}

func Test_runBranch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("衝突も汚れも無ければブランチを切って push しデフォルトブランチを移す", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()

			require.NoError(t, runBranch(f.runner(), []string{"-bump", "minor"}))
			assert.Equal(t, []string{
				fetchTagsCall,
				tagListCall,
				"git status --porcelain",
				"git fetch origin production",
				branchCreateCall,
				branchPushCall,
				defaultBranchCall,
			}, f.calls)
		})

		t.Run("接頭辞と分岐元はフラグで差し替えられる", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()

			require.NoError(t, runBranch(f.runner(), []string{"-bump", "patch", "-prefix", "hotfix", "-base", "staging"}))
			assert.Contains(t, f.calls, "git switch -c hotfix/v1.2.4 origin/staging")
			assert.Contains(t, f.calls, "gh repo edit --default-branch hotfix/v1.2.4")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できないフラグでは手順を 1 つも実行しない", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()

			err := runBranch(f.runner(), []string{"-no-such-flag"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse flags")
			assert.Empty(t, f.calls)
		})

		// タグ不在は tag 側と原因が同じでも復旧手順が違うため、案内を差し替える。
		t.Run("起点となるタグが無ければ初期タグ作成の案内へ差し替える", func(t *testing.T) {
			t.Parallel()

			err := runBranch((&fakeRunner{}).runner(), []string{"-bump", "patch"})
			require.ErrorIs(t, err, errNoTagForBranch)
			require.NotErrorIs(t, err, errNoTag)
		})

		// 差し替えるのはタグ不在だけ。取得そのものの失敗まで案内へ丸めると原因が消える。
		t.Run("タグの取得に失敗した場合は案内へ差し替えない", func(t *testing.T) {
			t.Parallel()
			f := &fakeRunner{failOn: tagListCall}

			err := runBranch(f.runner(), []string{"-bump", "patch"})
			require.ErrorIs(t, err, errFakeCommand)
			require.NotErrorIs(t, err, errNoTagForBranch)
		})

		t.Run("origin に同名ブランチが既にあればブランチを切らずに中止する", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()
			f.remoteBranches = map[string]bool{"release/v1.3.0": true}

			require.ErrorIs(t, runBranch(f.runner(), []string{"-bump", "minor"}), errBranchExists)
			assert.NotContains(t, f.calls, "git status --porcelain")
			assert.NotContains(t, f.calls, branchCreateCall)
		})

		t.Run("作業ツリーの状態を取得できなければブランチを切らない", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()
			f.failOn = "git status --porcelain"

			require.ErrorIs(t, runBranch(f.runner(), []string{"-bump", "minor"}), errFakeCommand)
			assert.NotContains(t, f.calls, branchCreateCall)
		})

		// 未コミットの変更を抱えたままブランチを切ると、切り替え先へ変更が付いて回る。
		t.Run("作業ツリーが汚れていれば中止し何が汚れているかを見せる", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()
			f.outputs["git status --porcelain"] = " M main.go\n"

			require.ErrorIs(t, runBranch(f.runner(), []string{"-bump", "minor"}), errDirtyWorktree)
			assert.Contains(t, f.calls, "git status --short")
			assert.NotContains(t, f.calls, branchCreateCall)
		})

		t.Run("ブランチを push できなければデフォルトブランチを移さない", func(t *testing.T) {
			t.Parallel()
			f := taggableRunner()
			f.failOn = branchPushCall

			require.ErrorIs(t, runBranch(f.runner(), []string{"-bump", "minor"}), errFakeCommand)
			assert.NotContains(t, f.calls, defaultBranchCall)
		})
	})
}

func Test_parseFlags(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグを解釈して値を取り込む", func(t *testing.T) {
			t.Parallel()
			fs := flag.NewFlagSet("tag", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			bumpKind := fs.String("bump", "", "")

			require.NoError(t, parseFlags(fs, []string{"-bump", "minor"}))
			assert.Equal(t, "minor", *bumpKind)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘルプ要求は通常のエラーと区別して返す", func(t *testing.T) {
			t.Parallel()
			fs := flag.NewFlagSet("tag", flag.ContinueOnError)
			fs.SetOutput(io.Discard)

			err := parseFlags(fs, []string{"-h"})

			require.ErrorIs(t, err, errHelpRequested)
		})

		t.Run("未知のフラグはヘルプ要求と混同せずエラーにする", func(t *testing.T) {
			t.Parallel()
			fs := flag.NewFlagSet("tag", flag.ContinueOnError)
			fs.SetOutput(io.Discard)

			err := parseFlags(fs, []string{"-bogus"})

			require.Error(t, err)
			assert.NotErrorIs(t, err, errHelpRequested)
		})
	})
}

//nolint:paralleltest // t.Chdir を使うため並列化不可
func Test_execute(t *testing.T) {
	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("tag はタグ作成の手順へ振り分ける", func(t *testing.T) {
			writeNote(t)
			f := taggableRunner()

			require.NoError(t, execute(f.runner(), []string{"tag", "-bump", "patch"}))
			assert.Contains(t, f.calls, annotateTagCall)
			assert.Contains(t, f.calls, tagPushCall)
			assert.Contains(t, f.calls, releaseCreateCall)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("branch はブランチ作成の手順へ振り分ける", func(t *testing.T) {
			f := taggableRunner()

			require.NoError(t, execute(f.runner(), []string{"branch", "-bump", "minor"}))
			assert.Contains(t, f.calls, branchCreateCall)
			assert.Contains(t, f.calls, branchPushCall)
			assert.Contains(t, f.calls, defaultBranchCall)
		})

		// ヘルプを異常終了にすると、同じリポジトリの他ツールと終了コードが食い違う。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("tag のヘルプ要求は失敗にせず手順を 1 つも実行しない", func(t *testing.T) {
			f := taggableRunner()

			require.NoError(t, execute(f.runner(), []string{"tag", "-h"}))
			assert.Empty(t, f.calls)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("branch のヘルプ要求は失敗にせず手順を 1 つも実行しない", func(t *testing.T) {
			f := taggableRunner()

			require.NoError(t, execute(f.runner(), []string{"branch", "-h"}))
			assert.Empty(t, f.calls)
		})
	})

	//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("サブコマンドが無ければ使い方を示して手順を 1 つも実行しない", func(t *testing.T) {
			f := taggableRunner()

			require.ErrorIs(t, execute(f.runner(), nil), errUsage)
			assert.Empty(t, f.calls)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("未知のサブコマンドでは手順を 1 つも実行しない", func(t *testing.T) {
			f := taggableRunner()

			require.ErrorIs(t, execute(f.runner(), []string{"bogus"}), errUnknownSubcommand)
			assert.Empty(t, f.calls)
		})

		// ヘルプ要求だけを飲み込む。失敗まで飲み込むと、タグを打てていないのに 0 で終わる。
		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("tag の失敗はヘルプ要求と区別してそのまま返す", func(t *testing.T) {
			require.ErrorIs(t, execute((&fakeRunner{}).runner(), []string{"tag", "-bump", "patch"}), errNoTag)
		})

		//nolint:paralleltest // 親が t.Chdir を使うため並列化不可
		t.Run("branch の失敗はヘルプ要求と区別してそのまま返す", func(t *testing.T) {
			require.ErrorIs(t, execute((&fakeRunner{}).runner(), []string{"branch", "-bump", "patch"}), errNoTagForBranch)
		})
	})
}
