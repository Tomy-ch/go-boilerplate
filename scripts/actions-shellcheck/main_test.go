package main

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go-boilerplate/scripts/lib/shellcheck"
)

const requireShellcheckEnv = "REQUIRE_SHELLCHECK"

const compositeAction = `name: sample
description: sample
runs:
  using: composite
  steps:
    - name: literal block
      shell: bash
      run: |
        echo hello
        echo world
    - name: plain scalar
      shell: bash
      run: echo plain
`

const aliasAction = `name: sample
runs:
  using: composite
  steps:
    - shell: bash
      run: &body |
        echo anchored
    - shell: bash
      run: *body
`

const mergeAction = `name: sample
runs:
  using: composite
  steps:
    - &base
      shell: bash
      run: echo merged
    - <<: *base
      name: second
`

const aliasDefectAction = `name: sample
runs:
  using: composite
  steps:
    - shell: bash
      run: &body |
        x="a b"
        echo $x
    - shell: bash
      run: *body
`

const mergeDefectAction = `name: sample
runs:
  using: composite
  steps:
    - &base
      shell: bash
      run: x="a b"; echo $x
    - <<: *base
      name: second
`

const quotedDefectAction = `runs:
  using: composite
  steps:
    - shell: bash
      run: 'x="a b"; echo $x'
`

const miscasedUsingAction = `runs:
  using: Composite
  steps:
    - shell: bash
      run: echo hi
`

var (
	// errWD は、作業ディレクトリの取得失敗の伝播を検証するためのセンチネルです。
	errWD = xerrors.New("getwd failed")
	// errLookPath は、shellcheck の所在確認の失敗を模すためのセンチネルです。
	errLookPath = xerrors.New("executable file not found in $PATH")
)

// failOpenFS は、指定した 1 パスの Open だけを失敗させる fs.FS。
//
// fstest.MapFS を埋め込むと ReadFileFS / ReadDirFS が昇格し、fs.ReadFile / fs.ReadDir が
// Open を経由しなくなって失敗を注入できないため、委譲で持つ。
type failOpenFS struct {
	fsys     fs.FS
	failPath string
}

func (f failOpenFS) Open(name string) (fs.File, error) {
	if name == f.failPath {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return f.fsys.Open(name)
}

func testFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for path, body := range files {
		fsys[path] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

// requireShellcheck は実物の shellcheck を要求する。手元に無い環境では skip するが、
// REQUIRE_SHELLCHECK を立てた実行では skip せず落とす（根拠は scripts/README.md の Notes）。
func requireShellcheck(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(shellcheck.Binary); err == nil {
		return
	}
	if os.Getenv(requireShellcheckEnv) != "" {
		t.Fatalf("shellcheck が PATH にありません（%s 指定時は skip しません）", requireShellcheckEnv)
	}
	t.Skip("shellcheck が PATH にありません")
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func parseDoc(t *testing.T, body string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(body), &doc))
	return &doc
}

func parseDocRoot(t *testing.T, body string) *yaml.Node {
	t.Helper()
	root := documentRoot(parseDoc(t, body))
	require.NotNil(t, root)
	return root
}

func symlinkFS(t *testing.T) fstest.MapFS {
	t.Helper()
	fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
	fsys[".github/actions/link/action.yml"] = &fstest.MapFile{
		Data: []byte("../real/action.yml"),
		Mode: fs.ModeSymlink,
	}
	fsys[".github/actions/link/dist.yml"] = &fstest.MapFile{
		Data: []byte("../real/action.yml"),
		Mode: fs.ModeSymlink,
	}
	fsys[".github/actions/dirlink"] = &fstest.MapFile{
		Data: []byte("real"),
		Mode: fs.ModeSymlink,
	}
	fsys[".github/actions/broken/action.yml"] = &fstest.MapFile{
		Data: []byte("../nowhere/action.yml"),
		Mode: fs.ModeSymlink,
	}
	return fsys
}

func Test_parseAction(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("literal ブロックの本文開始行はキー行の次を指す", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(compositeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, "echo hello\necho world\n", steps[0].script)
			assert.Equal(t, 9, steps[0].firstLine)
		})

		t.Run("plain スカラーの本文開始行はキー行そのものを指す", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(compositeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, "echo plain", steps[1].script)
			assert.Equal(t, 13, steps[1].firstLine)
		})

		t.Run("literal ブロックの列基準は本文のインデント幅になる", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(compositeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, 8, steps[0].colBase)
		})

		t.Run("plain スカラーの列基準は値の開始位置になる", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(compositeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, 11, steps[1].colBase)
		})

		t.Run("明示インデント指示子より深い本文でも列基準は剥がされた幅になる", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: |2\n          echo hi\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, "  echo hi\n", steps[0].script)
			assert.Equal(t, 8, steps[0].colBase)
		})

		t.Run("ダブルクォートのスカラーの列基準は開き引用符の内側を指す", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: \"echo hi\"\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, 12, steps[0].colBase)
		})

		t.Run("シングルクォートのスカラーの列基準は開き引用符の内側を指す", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: 'echo hi'\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, 12, steps[0].colBase)
		})

		t.Run("空行で始まる literal ブロックの列基準は最初の非空行のインデント幅になる", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: |\n\n        echo hi\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, 8, steps[0].colBase)
		})

		t.Run("alias で共有された run はアンカー先の本文を検査対象にする", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(aliasAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, "echo anchored\n", steps[1].script)
			assert.Equal(t, steps[0].firstLine, steps[1].firstLine)
		})

		t.Run("マージキーで継承した run と shell も抽出する", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte(mergeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, "echo merged", steps[1].script)
			assert.Equal(t, "bash", steps[1].shell)
		})

		t.Run("composite でない action は対象外", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte("runs:\n  using: node20\n  main: index.js\n"))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("トップレベルがマッピングでない YAML は対象 0 件", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte("- just\n- a list\n"))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("空ファイルは対象 0 件", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", nil)
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("steps キーを持たない composite は対象 0 件", func(t *testing.T) {
			t.Parallel()
			steps, err := parseAction("action.yaml", []byte("runs:\n  using: composite\n"))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("run を持たない composite は対象 0 件", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v7\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("shell が bash 以外でもステップとして抽出する", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: pwsh\n      run: Write-Host hi\n"
			steps, err := parseAction("action.yaml", []byte(body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, "pwsh", steps[0].shell)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shell 指定を欠く run ステップはエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - run: echo hi\n"
			_, err := parseAction("action.yaml", []byte(body))
			require.Error(t, err)
			require.ErrorIs(t, err, errNoShell)
			require.ErrorContains(t, err, "action.yaml:4")
		})

		t.Run("YAML として壊れていればエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := parseAction("action.yaml", []byte("runs: [\n"))
			require.Error(t, err)
			require.ErrorContains(t, err, "parse action.yaml")
		})

		t.Run("folded ブロックの run はエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: >\n        echo folded\n"
			_, err := parseAction("action.yaml", []byte(body))
			require.Error(t, err)
			require.ErrorIs(t, err, errFoldedRun)
			require.ErrorContains(t, err, "action.yaml:5")
		})

		t.Run("using の綴りが composite と一致しない action の run は件数差でエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := parseAction("action.yaml", []byte(miscasedUsingAction))
			require.Error(t, err)
			require.ErrorIs(t, err, errStepCountMismatch)
			require.ErrorContains(t, err, "抽出 0 / 期待 1")
		})

		t.Run("2 番目以降のドキュメントを黙って捨てずエラーにする", func(t *testing.T) {
			t.Parallel()
			body := compositeAction + "---\n" + compositeAction
			_, err := parseAction("action.yaml", []byte(body))
			require.Error(t, err)
			require.ErrorIs(t, err, errMultipleDocuments)
			require.ErrorContains(t, err, "action.yaml")
		})
	})
}

func Test_actionFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("action.yml と action.yaml をパス順に並べて返す", func(t *testing.T) {
			t.Parallel()
			files, err := actionFiles(testFS(map[string]string{
				".github/actions/z/action.yml":  compositeAction,
				".github/actions/a/action.yaml": compositeAction,
			}))
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/a/action.yaml", ".github/actions/z/action.yml"}, files)
		})

		t.Run("ネストした composite action も走査する", func(t *testing.T) {
			t.Parallel()
			files, err := actionFiles(testFS(map[string]string{
				".github/actions/group/sub/action.yaml": compositeAction,
			}))
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/group/sub/action.yaml"}, files)
		})

		t.Run("action 以外の YAML は対象にしない", func(t *testing.T) {
			t.Parallel()
			files, err := actionFiles(testFS(map[string]string{
				".github/actions/a/dist.yaml":   compositeAction,
				".github/actions/a/action.yaml": compositeAction,
			}))
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/a/action.yaml"}, files)
		})

		t.Run(".github/actions が無くてもエラーにしない", func(t *testing.T) {
			t.Parallel()
			files, err := actionFiles(testFS(map[string]string{"README.md": "# sample\n"}))
			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("action 定義ファイルへのシンボリックリンクも走査対象にする", func(t *testing.T) {
			t.Parallel()
			fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
			fsys[".github/actions/link/action.yml"] = &fstest.MapFile{
				Data: []byte("../real/action.yml"),
				Mode: fs.ModeSymlink,
			}
			files, err := actionFiles(fsys)
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/link/action.yml", ".github/actions/real/action.yml"}, files)
		})

		t.Run("action 定義以外へのシンボリックリンクは対象にしない", func(t *testing.T) {
			t.Parallel()
			fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
			fsys[".github/actions/link/dist.yml"] = &fstest.MapFile{
				Data: []byte("../real/action.yml"),
				Mode: fs.ModeSymlink,
			}
			files, err := actionFiles(fsys)
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/real/action.yml"}, files)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ディレクトリへのシンボリックリンクは黙って対象外にせずエラーにする", func(t *testing.T) {
			t.Parallel()
			fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
			fsys[".github/actions/link"] = &fstest.MapFile{
				Data: []byte("real"),
				Mode: fs.ModeSymlink,
			}
			_, err := actionFiles(fsys)
			require.Error(t, err)
			require.ErrorIs(t, err, errActionSymlinkDir)
		})

		t.Run("解決できないシンボリックリンクは黙って対象外にせずエラーにする", func(t *testing.T) {
			t.Parallel()
			fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
			fsys[".github/actions/link/action.yml"] = &fstest.MapFile{
				Data: []byte("../nowhere/action.yml"),
				Mode: fs.ModeSymlink,
			}
			_, err := actionFiles(fsys)
			require.Error(t, err)
			require.ErrorIs(t, err, errActionSymlinkUnresolved)
		})

		t.Run("走査中の読み取り失敗は対象外に寄せずエラーにする", func(t *testing.T) {
			t.Parallel()
			fsys := failOpenFS{
				fsys:     testFS(map[string]string{".github/actions/a/action.yaml": compositeAction}),
				failPath: ".github/actions/a",
			}

			_, err := actionFiles(fsys)
			require.ErrorIs(t, err, fs.ErrPermission)
			require.ErrorContains(t, err, "walk "+actionsDir)
		})
	})
}

func Test_collectSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("action.yml と action.yaml の両方から抽出する", func(t *testing.T) {
			t.Parallel()
			files, steps, err := collectSteps(testFS(map[string]string{
				".github/actions/a/action.yaml": compositeAction,
				".github/actions/b/action.yml":  compositeAction,
			}))
			require.NoError(t, err)
			assert.Len(t, files, 2)
			assert.Len(t, steps, 4)
		})

		t.Run("composite action が無ければ 0 件を返す", func(t *testing.T) {
			t.Parallel()
			files, steps, err := collectSteps(testFS(map[string]string{
				".github/workflows/ci.yaml": "on: push\njobs: {}\n",
			}))
			require.NoError(t, err)
			assert.Empty(t, files)
			assert.Empty(t, steps)
		})

		t.Run("run を持たない composite action だけなら 0 件でもエラーにしない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v7\n"
			_, steps, err := collectSteps(testFS(map[string]string{".github/actions/a/action.yaml": body}))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("composite でない action の run: 文字列は抽出破損とみなさない", func(t *testing.T) {
			t.Parallel()
			body := "inputs:\n  run:\n    description: command to run\nruns:\n  using: node20\n  main: index.js\n"
			_, steps, err := collectSteps(testFS(map[string]string{".github/actions/a/action.yaml": body}))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("steps をマッピングで書いた action はエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps: {}\n  extra:\n    - run: echo hi\n"
			_, _, err := collectSteps(testFS(map[string]string{".github/actions/a/action.yaml": body}))
			require.Error(t, err)
			require.ErrorIs(t, err, errStepsNotSequence)
		})

		t.Run("他のファイルが健全でも 1 ファイルの抽出破損はエラーにする", func(t *testing.T) {
			t.Parallel()
			_, _, err := collectSteps(testFS(map[string]string{
				".github/actions/a/action.yaml": compositeAction,
				".github/actions/b/action.yaml": miscasedUsingAction,
			}))
			require.Error(t, err)
			require.ErrorIs(t, err, errStepCountMismatch)
			require.ErrorContains(t, err, ".github/actions/b/action.yaml")
		})

		t.Run("走査そのものが失敗した場合は 0 件で返さずエラーにする", func(t *testing.T) {
			t.Parallel()
			fsys := testFS(map[string]string{".github/actions/real/action.yml": compositeAction})
			fsys[".github/actions/link"] = &fstest.MapFile{
				Data: []byte("real"),
				Mode: fs.ModeSymlink,
			}

			files, steps, err := collectSteps(fsys)
			require.ErrorIs(t, err, errActionSymlinkDir)
			assert.Nil(t, files)
			assert.Nil(t, steps)
		})

		t.Run("走査できた action 定義ファイルを読めない場合はエラーにする", func(t *testing.T) {
			t.Parallel()
			fsys := failOpenFS{
				fsys:     testFS(map[string]string{".github/actions/a/action.yaml": compositeAction}),
				failPath: ".github/actions/a/action.yaml",
			}

			_, _, err := collectSteps(fsys)
			require.ErrorIs(t, err, fs.ErrPermission)
			require.ErrorContains(t, err, "read .github/actions/a/action.yaml")
		})
	})
}

func Test_countRunSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("using の値と無関係に run ステップを数える", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: COMPOSITE\n  steps:\n    - shell: bash\n      run: echo hi\n"
			count, err := countRunSteps("action.yaml", []byte(body))
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		})

		t.Run("alias で継承した run も数える", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", []byte(aliasAction))
			require.NoError(t, err)
			assert.Equal(t, 2, count)
		})

		t.Run("マージキーで継承した run も数える", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", []byte(mergeAction))
			require.NoError(t, err)
			assert.Equal(t, 2, count)
		})

		t.Run("run を持たないステップは数えない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v7\n"
			count, err := countRunSteps("action.yaml", []byte(body))
			require.NoError(t, err)
			assert.Zero(t, count)
		})

		t.Run("run が文字列でないステップは数えない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: 123\n"
			count, err := countRunSteps("action.yaml", []byte(body))
			require.NoError(t, err)
			assert.Zero(t, count)
		})

		t.Run("トップレベルがマッピングでない YAML は 0 件", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", []byte("- just\n- a list\n"))
			require.NoError(t, err)
			assert.Zero(t, count)
		})

		t.Run("空ファイルは 0 件", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", nil)
			require.NoError(t, err)
			assert.Zero(t, count)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("steps がリストとして読めなければエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps: {}\n"
			_, err := countRunSteps("action.yaml", []byte(body))
			require.Error(t, err)
			require.ErrorIs(t, err, errStepsNotSequence)
			require.ErrorContains(t, err, "action.yaml")
		})

		t.Run("デコードできない YAML は 0 件で返さずエラーにする", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", []byte("runs: [\n"))
			require.Error(t, err)
			require.ErrorContains(t, err, "decode action.yaml")
			assert.Zero(t, count)
		})

		t.Run("alias が自分自身を含む YAML は 0 件で返さずエラーにする", func(t *testing.T) {
			t.Parallel()
			count, err := countRunSteps("action.yaml", []byte("runs: &r\n  steps: *r\n"))
			require.Error(t, err)
			require.ErrorContains(t, err, "decode action.yaml")
			assert.Zero(t, count)
		})
	})
}

func Test_blockIndentWidth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空行を読み飛ばして最初の非空行のインデント幅を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 8, blockIndentWidth([]byte("run: |\n\n        echo hi\n"), 2, "\necho hi\n"))
		})

		t.Run("値に残ったインデントの分を差し引く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 8, blockIndentWidth([]byte("run: |2\n          echo hi\n"), 2, "  echo hi\n"))
		})

		t.Run("本文が空行だけなら 0 を返す", func(t *testing.T) {
			t.Parallel()
			assert.Zero(t, blockIndentWidth([]byte("run: |\n\n\n"), 2, "\n\n"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本文開始行が行範囲の外なら 0 を返す", func(t *testing.T) {
			t.Parallel()
			data := []byte("run: |\n        echo hi\n")
			assert.Zero(t, blockIndentWidth(data, 0, "echo hi\n"))
			assert.Zero(t, blockIndentWidth(data, 99, "echo hi\n"))
		})
	})
}

func Test_shellDialect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shellcheck が扱える shell は方言名へ写す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", shellDialect("bash"))
			assert.Equal(t, "bash", shellDialect("bash -e {0}"))
			assert.Equal(t, "bash", shellDialect("/usr/bin/bash --noprofile {0}"))
			assert.Equal(t, "sh", shellDialect("sh"))
		})

		t.Run("env 経由の指定でも後続コマンドを方言として扱う", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", shellDialect("/usr/bin/env bash"))
			assert.Equal(t, "bash", shellDialect("env bash {0}"))
		})

		t.Run("env の前置き変数代入を読み飛ばして方言を採る", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", shellDialect("env FOO=bar bash"))
			assert.Equal(t, "bash", shellDialect("/usr/bin/env FOO=bar BAZ=qux bash {0}"))
		})

		t.Run("数字を含む変数名の前置きでも読み飛ばして方言を採る", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", shellDialect("env GO111MODULE=on bash"))
			assert.Equal(t, "bash", shellDialect("env _X2=1 bash"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shell が pwsh なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("pwsh")]
			assert.False(t, ok)
		})

		t.Run("shell が powershell なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("powershell")]
			assert.False(t, ok)
		})

		t.Run("shell が python なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("python")]
			assert.False(t, ok)
		})

		t.Run("shell が cmd なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("cmd")]
			assert.False(t, ok)
		})

		t.Run("shell が env のみ なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("env")]
			assert.False(t, ok)
		})

		t.Run("shell が env と変数代入だけ なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("env FOO=bar")]
			assert.False(t, ok)
		})

		t.Run("先頭が数字の名前は変数代入とみなさず方言判定をそこで打ち切る", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "2FOO=bar", shellDialect("env 2FOO=bar bash"))
			_, ok := shebangs[shellDialect("env 2FOO=bar bash")]
			assert.False(t, ok)
		})

		t.Run("shell が 空 なら方言なしとして扱う", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("")]
			assert.False(t, ok)
		})

		t.Run("式で指定された shell は方言を決められないため対象外にする", func(t *testing.T) {
			t.Parallel()
			_, ok := shebangs[shellDialect("${{ inputs.shell }}")]
			assert.False(t, ok)
		})
	})
}

func Test_maskExpressions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("式をプレースホルダへ置換する", func(t *testing.T) {
			t.Parallel()
			masked, err := maskExpressions(`echo "${{ inputs.name }}"`)
			require.NoError(t, err)
			assert.Equal(t, `echo "GH_EXPR"`, masked)
		})

		t.Run("同一行に複数あっても式の間のコマンドを残す", func(t *testing.T) {
			t.Parallel()
			masked, err := maskExpressions("echo ${{ inputs.a }} important_command ${{ inputs.b }}")
			require.NoError(t, err)
			assert.Equal(t, "echo GH_EXPR important_command GH_EXPR", masked)
		})

		t.Run("複数行にまたがる式でも行数を保つ", func(t *testing.T) {
			t.Parallel()
			masked, err := maskExpressions("echo ${{ inputs.a\n  || inputs.b }}\necho last")
			require.NoError(t, err)
			assert.Equal(t, "echo GH_EXPR\n\necho last", masked)
			assert.Equal(t, 2, strings.Count(masked, "\n"))
		})

		t.Run("クォート内の }} で式を打ち切らない", func(t *testing.T) {
			t.Parallel()
			masked, err := maskExpressions(`echo "${{ contains(github.event.body, '}}') }}"`)
			require.NoError(t, err)
			assert.Equal(t, `echo "GH_EXPR"`, masked)
		})

		t.Run("式を含まない本文は変えない", func(t *testing.T) {
			t.Parallel()
			masked, err := maskExpressions("echo ${HOME}")
			require.NoError(t, err)
			assert.Equal(t, "echo ${HOME}", masked)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("閉じていない式は本文を切り捨てずエラーにする", func(t *testing.T) {
			t.Parallel()
			_, err := maskExpressions("echo ${{ inputs.a\nrm -rf /\n")
			require.Error(t, err)
			require.ErrorIs(t, err, errUnterminatedExpr)
		})

		t.Run("式内のクォートが奇数個でも後続の式までのシェルを飲み込まない", func(t *testing.T) {
			t.Parallel()
			_, err := maskExpressions("echo ${{ inputs.msg == 'it's ok' }}\nrm -rf /\necho ${{ inputs.done }}\n")
			require.Error(t, err)
			require.ErrorIs(t, err, errUnterminatedExpr)
		})
	})
}

func Test_exprEnd(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最初の閉じ位置を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 3, exprEnd(" a }} rest"))
		})

		t.Run("クォート内の閉じ記号は無視する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 9, exprEnd(" a('}}') }}"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("閉じが無ければ -1 を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, -1, exprEnd(" a "))
		})

		t.Run("自身の閉じより先に次の式が始まれば -1 を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, -1, exprEnd(" 'a }} rm -rf / ${{ 'b }}"))
		})
	})
}

func Test_remapFindings(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shellcheck の行番号を action ファイルの行番号へ写す", func(t *testing.T) {
			t.Parallel()
			s := step{file: ".github/actions/a/action.yaml", firstLine: 9}
			out := "-:3:15: note: Double quote to prevent globbing [SC2086]\n"
			findings := remapFindings(s, out)
			require.Len(t, findings, 1)
			assert.Contains(t, findings[0], ".github/actions/a/action.yaml:10:15:")
			assert.Contains(t, findings[0], "[SC2086]")
		})

		t.Run("列番号は本文のインデント幅だけずらす", func(t *testing.T) {
			t.Parallel()
			s := step{file: "action.yaml", firstLine: 9, colBase: 8}
			findings := remapFindings(s, "-:3:15: note: msg [SC2086]\n")
			require.Len(t, findings, 1)
			assert.Contains(t, findings[0], "action.yaml:10:23:")
		})

		t.Run("本文 1 行目の指摘は本文開始行を指す", func(t *testing.T) {
			t.Parallel()
			s := step{file: "action.yaml", firstLine: 9}
			findings := remapFindings(s, "-:2:1: note: msg [SC1000]\n")
			require.Len(t, findings, 1)
			assert.Contains(t, findings[0], "action.yaml:9:1:")
		})

		t.Run("shebang 行の指摘は run キー行を指す", func(t *testing.T) {
			t.Parallel()
			s := step{file: "action.yaml", firstLine: 9}
			findings := remapFindings(s, "-:1:1: error: msg [SC1008]\n")
			require.Len(t, findings, 1)
			assert.Contains(t, findings[0], "action.yaml:8:1:")
		})

		t.Run("指摘なしなら空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, remapFindings(step{file: "action.yaml", firstLine: 1}, ""))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解析できない行は無視する", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, remapFindings(step{file: "action.yaml", firstLine: 1}, "unexpected output\n"))
		})

		t.Run("行番号が整数として読めない行は無視する", func(t *testing.T) {
			t.Parallel()
			out := "-:99999999999999999999:15: note: msg [SC2086]\n"
			assert.Empty(t, remapFindings(step{file: "action.yaml", firstLine: 9}, out))
		})

		t.Run("列番号が整数として読めない行は無視する", func(t *testing.T) {
			t.Parallel()
			out := "-:3:99999999999999999999: note: msg [SC2086]\n"
			assert.Empty(t, remapFindings(step{file: "action.yaml", firstLine: 9}, out))
		})

		t.Run("読める行と読めない行が混ざれば読める行だけを返す", func(t *testing.T) {
			t.Parallel()
			out := "-:99999999999999999999:15: note: broken [SC2086]\n-:3:15: note: msg [SC2086]\n"

			findings := remapFindings(step{file: "action.yaml", firstLine: 9}, out)

			require.Len(t, findings, 1)
			assert.Contains(t, findings[0], "action.yaml:10:15:")
		})
	})
}

func Test_check(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象外 shell は検査せず skip として報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			steps := []step{
				{file: "action.yaml", shell: "bash", script: "echo ok\n", firstLine: 1},
				{file: "action.yaml", shell: "pwsh", script: "Write-Host hi\n", firstLine: 5},
			}
			res, err := check(t.Context(), steps)
			require.NoError(t, err)
			assert.Equal(t, 1, res.checked)
			require.Len(t, res.skipped, 1)
			assert.Contains(t, res.skipped[0], "action.yaml:5")
			assert.Contains(t, res.skipped[0], "pwsh")
			assert.Empty(t, res.findings)
		})

		t.Run("実 action と同じ形の本文から指摘を行番号付きで返す", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			steps, err := parseAction("action.yaml", []byte(strings.Replace(
				compositeAction, "        echo world\n", "        x=\"a b\"; echo $x\n", 1)))
			require.NoError(t, err)
			res, err := check(t.Context(), steps)
			require.NoError(t, err)
			assert.Equal(t, 2, res.checked)
			require.Len(t, res.findings, 1)
			assert.Contains(t, res.findings[0], "action.yaml:10:")
			assert.Contains(t, res.findings[0], "SC2086")
		})

		t.Run("alias で共有された run も shellcheck に掛けアンカー先の位置で報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			_, steps, err := collectSteps(testFS(map[string]string{
				".github/actions/a/action.yaml": aliasDefectAction,
			}))
			require.NoError(t, err)
			res, err := check(t.Context(), steps)
			require.NoError(t, err)
			assert.Equal(t, 2, res.checked)
			require.Len(t, res.findings, 2)
			for _, finding := range res.findings {
				assert.Contains(t, finding, ".github/actions/a/action.yaml:8:14:")
				assert.Contains(t, finding, "SC2086")
			}
		})

		t.Run("引用符付きスカラーの run も列位置ごと報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			_, steps, err := collectSteps(testFS(map[string]string{
				".github/actions/a/action.yaml": quotedDefectAction,
			}))
			require.NoError(t, err)
			res, err := check(t.Context(), steps)
			require.NoError(t, err)
			assert.Equal(t, 1, res.checked)
			require.Len(t, res.findings, 1)
			assert.Contains(t, res.findings[0], ".github/actions/a/action.yaml:5:27:")
			assert.Contains(t, res.findings[0], "SC2086")
		})

		t.Run("マージキーで継承した plain スカラーの run も列位置ごと報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			_, steps, err := collectSteps(testFS(map[string]string{
				".github/actions/a/action.yaml": mergeDefectAction,
			}))
			require.NoError(t, err)
			res, err := check(t.Context(), steps)
			require.NoError(t, err)
			assert.Equal(t, 2, res.checked)
			require.Len(t, res.findings, 2)
			for _, finding := range res.findings {
				assert.Contains(t, finding, ".github/actions/a/action.yaml:7:26:")
				assert.Contains(t, finding, "SC2086")
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shellcheck の失敗はそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			steps := []step{{file: "action.yaml", shell: "bash", script: "echo ok\n", firstLine: 1}}
			res, err := check(canceledContext(t), steps)
			require.Error(t, err)
			require.ErrorIs(t, err, shellcheck.ErrRun)
			assert.Zero(t, res.checked)
			assert.Empty(t, res.findings)
		})

		t.Run("閉じていない式は検査せずエラーにする", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			steps := []step{{file: "action.yaml", shell: "bash", script: "echo ${{ inputs.a\n", firstLine: 9}}
			_, err := check(t.Context(), steps)
			require.Error(t, err)
			require.ErrorIs(t, err, errUnterminatedExpr)
			require.ErrorContains(t, err, "action.yaml:9")
		})
	})
}

func Test_appendSymlink(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("action 定義ファイルへのリンクは既に集めたパスを保ったまま追記する", func(t *testing.T) {
			t.Parallel()
			files := []string{".github/actions/first/action.yml"}
			err := appendSymlink(symlinkFS(t), ".github/actions/link/action.yml", "action.yml", &files)
			require.NoError(t, err)
			assert.Equal(t, []string{
				".github/actions/first/action.yml",
				".github/actions/link/action.yml",
			}, files)
		})

		t.Run("action 定義でない名前のリンクは追記しない", func(t *testing.T) {
			t.Parallel()
			files := []string{".github/actions/first/action.yml"}
			err := appendSymlink(symlinkFS(t), ".github/actions/link/dist.yml", "dist.yml", &files)
			require.NoError(t, err)
			assert.Equal(t, []string{".github/actions/first/action.yml"}, files)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ディレクトリへのリンクは対象のパスを添えてエラーにする", func(t *testing.T) {
			t.Parallel()
			files := []string{".github/actions/first/action.yml"}
			err := appendSymlink(symlinkFS(t), ".github/actions/dirlink", "dirlink", &files)
			require.ErrorIs(t, err, errActionSymlinkDir)
			require.ErrorContains(t, err, ".github/actions/dirlink")
			assert.Equal(t, []string{".github/actions/first/action.yml"}, files)
		})

		t.Run("解決できないリンクは対象のパスを添えてエラーにする", func(t *testing.T) {
			t.Parallel()
			files := []string{".github/actions/first/action.yml"}
			err := appendSymlink(symlinkFS(t), ".github/actions/broken/action.yml", "action.yml", &files)
			require.ErrorIs(t, err, errActionSymlinkUnresolved)
			require.ErrorContains(t, err, ".github/actions/broken/action.yml")
			assert.Equal(t, []string{".github/actions/first/action.yml"}, files)
		})
	})
}

func Test_isActionFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("action.yml は action 定義ファイルとして扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isActionFile("action.yml"))
		})

		t.Run("action.yaml は action 定義ファイルとして扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isActionFile("action.yaml"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接尾辞が一致するだけの名前は対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isActionFile("my-action.yaml"))
		})

		t.Run("拡張子の後ろに続きがある名前は対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isActionFile("action.yaml.bak"))
		})

		t.Run("action と綴りが異なる名前は対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isActionFile("actions.yaml"))
		})

		t.Run("空の名前は対象にしない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isActionFile(""))
		})
	})
}

func Test_requireSingleDocument(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一ドキュメントならエラーにしない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, requireSingleDocument("action.yaml", []byte(compositeAction)))
		})

		t.Run("空ファイルならエラーにしない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, requireSingleDocument("action.yaml", nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2 番目のドキュメントがあればファイル名を添えてエラーにする", func(t *testing.T) {
			t.Parallel()
			err := requireSingleDocument("action.yaml", []byte("name: a\n---\nname: b\n"))
			require.ErrorIs(t, err, errMultipleDocuments)
			require.ErrorContains(t, err, "action.yaml")
		})

		t.Run("1 番目のドキュメントが壊れていればエラーにする", func(t *testing.T) {
			t.Parallel()
			err := requireSingleDocument("action.yaml", []byte("runs: [\n"))
			require.Error(t, err)
			require.ErrorContains(t, err, "parse action.yaml")
		})

		t.Run("2 番目のドキュメントが壊れていればエラーにする", func(t *testing.T) {
			t.Parallel()
			err := requireSingleDocument("action.yaml", []byte("name: a\n---\nruns: [\n"))
			require.Error(t, err)
			require.ErrorContains(t, err, "parse action.yaml")
		})
	})
}

func Test_fieldValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マッピングからキーに対応する値を返す", func(t *testing.T) {
			t.Parallel()
			node := map[string]any{"runs": "composite", "name": "sample"}
			assert.Equal(t, "composite", fieldValue(node, "runs"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないキーは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, fieldValue(map[string]any{"name": "sample"}, "runs"))
		})

		t.Run("マッピングでない値は nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, fieldValue([]any{"runs"}, "runs"))
		})

		t.Run("nil は nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, fieldValue(nil, "runs"))
		})
	})
}

func Test_extractSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("抽出したステップは指摘の写し戻し先となる対象ファイル名を持つ", func(t *testing.T) {
			t.Parallel()
			file := ".github/actions/a/action.yaml"
			steps, err := extractSteps(file, []byte(compositeAction), parseDoc(t, compositeAction))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, file, steps[0].file)
			assert.Equal(t, file, steps[1].file)
		})

		t.Run("using が composite でない action は抽出せずエラーにもしない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: node20\n  steps:\n    - shell: bash\n      run: echo hi\n"
			steps, err := extractSteps("action.yaml", []byte(body), parseDoc(t, body))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("using が無い action は抽出せずエラーにもしない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  steps:\n    - shell: bash\n      run: echo hi\n"
			steps, err := extractSteps("action.yaml", []byte(body), parseDoc(t, body))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("steps を持たない composite は抽出せずエラーにもしない", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n"
			steps, err := extractSteps("action.yaml", []byte(body), parseDoc(t, body))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("run を持たないステップは読み飛ばして後続の run を抽出する", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v7\n" +
				"    - shell: bash\n      run: echo hi\n"
			steps, err := extractSteps("action.yaml", []byte(body), parseDoc(t, body))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, "echo hi", steps[0].script)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shell を欠くステップがあれば後続が健全でも 1 件も返さずエラーにする", func(t *testing.T) {
			t.Parallel()
			body := "runs:\n  using: composite\n  steps:\n    - run: echo first\n" +
				"    - shell: bash\n      run: echo second\n"
			steps, err := extractSteps("action.yaml", []byte(body), parseDoc(t, body))
			require.ErrorIs(t, err, errNoShell)
			assert.Empty(t, steps)
		})
	})
}

func Test_documentRoot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドキュメントノードの最初の要素を返す", func(t *testing.T) {
			t.Parallel()
			root := documentRoot(parseDoc(t, "runs:\n  using: composite\n"))
			require.NotNil(t, root)
			assert.Equal(t, yaml.MappingNode, root.Kind)
			assert.Equal(t, "runs", root.Content[0].Value)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドキュメントノードでなければ nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, documentRoot(&yaml.Node{Kind: yaml.MappingNode}))
		})

		t.Run("内容の無いドキュメントは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, documentRoot(&yaml.Node{Kind: yaml.DocumentNode}))
		})
	})
}

func Test_mapValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーに対応する値ノードを返す", func(t *testing.T) {
			t.Parallel()
			v := mapValue(parseDocRoot(t, "using: composite\nname: sample\n"), "using")
			require.NotNil(t, v)
			assert.Equal(t, "composite", v.Value)
		})

		t.Run("値がエイリアスならアンカー先を解決して返す", func(t *testing.T) {
			t.Parallel()
			v := mapValue(parseDocRoot(t, "a: &anchor composite\nb: *anchor\n"), "b")
			require.NotNil(t, v)
			assert.Equal(t, "composite", v.Value)
		})

		t.Run("マッピング自体がエイリアスでも解決して引ける", func(t *testing.T) {
			t.Parallel()
			root := parseDocRoot(t, "base: &base\n  shell: sh\nuse: *base\n")
			v := mapValue(mapValue(root, "use"), "shell")
			require.NotNil(t, v)
			assert.Equal(t, "sh", v.Value)
		})

		t.Run("直接書かれたキーはマージキー経由の値より優先する", func(t *testing.T) {
			t.Parallel()
			root := parseDocRoot(t, "base: &base\n  shell: sh\nstep:\n  <<: *base\n  shell: bash\n")
			v := mapValue(mapValue(root, "step"), "shell")
			require.NotNil(t, v)
			assert.Equal(t, "bash", v.Value)
		})

		t.Run("直接書かれたキーが無ければマージキー経由で引く", func(t *testing.T) {
			t.Parallel()
			root := parseDocRoot(t, "base: &base\n  shell: sh\nstep:\n  <<: *base\n  name: second\n")
			v := mapValue(mapValue(root, "step"), "shell")
			require.NotNil(t, v)
			assert.Equal(t, "sh", v.Value)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないキーは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, mapValue(parseDocRoot(t, "using: composite\n"), "steps"))
		})

		t.Run("マッピングでないノードは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, mapValue(parseDocRoot(t, "- a\n- b\n"), "using"))
		})

		t.Run("nil ノードは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, mapValue(nil, "using"))
		})
	})
}

func Test_mergeValue(t *testing.T) {
	t.Parallel()

	shellMapping := func(value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "shell"},
			{Kind: yaml.ScalarNode, Value: value},
		}}
	}
	nameMapping := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "name"},
		{Kind: yaml.ScalarNode, Value: "second"},
	}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マージ元がマッピングならそのキーを引く", func(t *testing.T) {
			t.Parallel()
			v := mergeValue(shellMapping("sh"), "shell")
			require.NotNil(t, v)
			assert.Equal(t, "sh", v.Value)
		})

		t.Run("マージ元がリストなら先に並んだ側を優先する", func(t *testing.T) {
			t.Parallel()
			seq := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				shellMapping("sh"), shellMapping("bash"),
			}}
			v := mergeValue(seq, "shell")
			require.NotNil(t, v)
			assert.Equal(t, "sh", v.Value)
		})

		t.Run("リストの先頭にキーが無ければ後続から引く", func(t *testing.T) {
			t.Parallel()
			seq := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				nameMapping, shellMapping("bash"),
			}}
			v := mergeValue(seq, "shell")
			require.NotNil(t, v)
			assert.Equal(t, "bash", v.Value)
		})

		t.Run("マージ元がエイリアスでも解決して引く", func(t *testing.T) {
			t.Parallel()
			alias := &yaml.Node{Kind: yaml.AliasNode, Alias: shellMapping("sh")}
			v := mergeValue(alias, "shell")
			require.NotNil(t, v)
			assert.Equal(t, "sh", v.Value)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マージ元が無ければ nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, mergeValue(nil, "shell"))
		})

		t.Run("リストのどこにもキーが無ければ nil を返す", func(t *testing.T) {
			t.Parallel()
			seq := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{nameMapping}}
			assert.Nil(t, mergeValue(seq, "shell"))
		})

		t.Run("マッピングでもリストでもないノードは nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, mergeValue(&yaml.Node{Kind: yaml.ScalarNode, Value: "sh"}, "shell"))
		})
	})
}

func Test_resolveAlias(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エイリアスの連鎖を辿って実体のノードを返す", func(t *testing.T) {
			t.Parallel()
			target := &yaml.Node{Kind: yaml.ScalarNode, Value: "echo hi"}
			outer := &yaml.Node{Kind: yaml.AliasNode, Alias: &yaml.Node{Kind: yaml.AliasNode, Alias: target}}
			assert.Same(t, target, resolveAlias(outer))
		})

		t.Run("エイリアスでないノードはそのまま返す", func(t *testing.T) {
			t.Parallel()
			target := &yaml.Node{Kind: yaml.ScalarNode, Value: "echo hi"}
			assert.Same(t, target, resolveAlias(target))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil は nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, resolveAlias(nil))
		})
	})
}

func Test_bodyFirstLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("literal ブロックの本文はキー行の次の行から始まる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 10, bodyFirstLine(&yaml.Node{Style: yaml.LiteralStyle, Line: 9}))
		})

		t.Run("plain スカラーの本文はキー行そのものから始まる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 9, bodyFirstLine(&yaml.Node{Line: 9}))
		})

		t.Run("引用符付きスカラーの本文はキー行そのものから始まる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 9, bodyFirstLine(&yaml.Node{Style: yaml.DoubleQuotedStyle, Line: 9}))
		})
	})
}

func Test_bodyColumnBase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("literal ブロックはキー行の列でなく剥がされたインデント幅を基準にする", func(t *testing.T) {
			t.Parallel()
			data := []byte("      run: |\n        echo hi\n")
			run := &yaml.Node{Style: yaml.LiteralStyle, Column: 12, Value: "echo hi\n"}
			assert.Equal(t, 8, bodyColumnBase(data, run, 2))
		})

		t.Run("シングルクォートのスカラーは開き引用符の分だけ進めた位置を基準にする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 12, bodyColumnBase(nil, &yaml.Node{Style: yaml.SingleQuotedStyle, Column: 12}, 1))
		})

		t.Run("ダブルクォートのスカラーは開き引用符の分だけ進めた位置を基準にする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 12, bodyColumnBase(nil, &yaml.Node{Style: yaml.DoubleQuotedStyle, Column: 12}, 1))
		})

		t.Run("plain スカラーは値の開始位置の 1 つ手前を基準にする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 11, bodyColumnBase(nil, &yaml.Node{Column: 12}, 1))
		})
	})
}

func Test_firstIndentWidth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最初の非空行のインデント幅を返す", func(t *testing.T) {
			t.Parallel()
			width, ok := firstIndentWidth([]string{"    echo hi", "  echo bye"})
			require.True(t, ok)
			assert.Equal(t, 4, width)
		})

		t.Run("空行と空白だけの行は読み飛ばす", func(t *testing.T) {
			t.Parallel()
			width, ok := firstIndentWidth([]string{"", "   ", "  echo hi"})
			require.True(t, ok)
			assert.Equal(t, 2, width)
		})

		t.Run("タブもインデント幅として数える", func(t *testing.T) {
			t.Parallel()
			width, ok := firstIndentWidth([]string{"\t\techo hi"})
			require.True(t, ok)
			assert.Equal(t, 2, width)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非空行が無ければ幅なしとして返す", func(t *testing.T) {
			t.Parallel()
			width, ok := firstIndentWidth([]string{"", "  ", "\t"})
			assert.False(t, ok)
			assert.Zero(t, width)
		})

		t.Run("行が 1 つも無ければ幅なしとして返す", func(t *testing.T) {
			t.Parallel()
			width, ok := firstIndentWidth(nil)
			assert.False(t, ok)
			assert.Zero(t, width)
		})
	})
}

func Test_isAssignment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("英字で始まる名前の代入は代入として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isAssignment("FOO=bar"))
		})

		t.Run("アンダースコア始まりで数字を含む名前も代入として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isAssignment("_X2=1"))
		})

		t.Run("値が空の代入も代入として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isAssignment("FOO="))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("= を含まない語は代入として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isAssignment("bash"))
		})

		t.Run("名前が空の語は代入として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isAssignment("=bar"))
		})

		t.Run("数字で始まる名前は代入として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isAssignment("2FOO=bar"))
		})

		t.Run("名前に記号を含む語は代入として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isAssignment("FOO-BAR=1"))
		})
	})
}

func Test_fieldBase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パス区切りの末尾だけを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", fieldBase("/usr/bin/bash"))
		})

		t.Run("区切りが無ければそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "bash", fieldBase("bash"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区切りで終わる場合は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, fieldBase("/usr/bin/"))
		})

		t.Run("空文字はそのまま空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, fieldBase(""))
		})
	})
}

// stubWD は、固定のディレクトリを返す作業ディレクトリの取得手段です。
func stubWD(root string) func() (string, error) {
	return func() (string, error) { return root, nil }
}

// stubLookPath は、shellcheck の所在確認の結果を固定する取得手段です。
func stubLookPath(err error) func(string) (string, error) {
	return func(name string) (string, error) { return name, err }
}

// writeActionRoot は action 定義 1 件を含む作業ディレクトリを作り、その絶対パスを返します。
func writeActionRoot(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, actionsDir, "sample", "action.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return root
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象外の shell しか無ければ shellcheck を呼ばずに成功する", func(t *testing.T) {
			t.Parallel()
			root := writeActionRoot(t, "runs:\n  using: composite\n  steps:\n    - shell: python\n      run: print(1)\n")

			require.NoError(t, run(t.Context(), stubWD(root), stubLookPath(nil)))
		})

		t.Run("指摘の無いスクリプトは成功する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			root := writeActionRoot(t, compositeAction)

			require.NoError(t, run(t.Context(), stubWD(root), stubLookPath(nil)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shellcheck が PATH に無ければ走査へ進まず失敗する", func(t *testing.T) {
			t.Parallel()

			err := run(t.Context(), stubWD(writeActionRoot(t, compositeAction)), stubLookPath(errLookPath))

			require.ErrorIs(t, err, errShellcheckMissing)
			assert.ErrorContains(t, err, errLookPath.Error())
		})

		t.Run("作業ディレクトリを取得できなければ失敗する", func(t *testing.T) {
			t.Parallel()

			err := run(t.Context(), func() (string, error) { return "", errWD }, stubLookPath(nil))

			require.ErrorIs(t, err, errWD)
			assert.ErrorContains(t, err, "getwd")
		})

		t.Run("action 定義を解釈できなければ失敗する", func(t *testing.T) {
			t.Parallel()
			root := writeActionRoot(t, "runs:\n  using: composite\n  steps: 1\n")

			err := run(t.Context(), stubWD(root), stubLookPath(nil))

			require.ErrorIs(t, err, errStepsNotSequence)
		})

		t.Run("検査に掛けられないステップがあれば失敗する", func(t *testing.T) {
			t.Parallel()
			root := writeActionRoot(t, "runs:\n  using: composite\n  steps:\n    - shell: bash\n      run: echo ${{ inputs.a\n")

			err := run(t.Context(), stubWD(root), stubLookPath(nil))

			require.ErrorIs(t, err, errUnterminatedExpr)
		})

		t.Run("指摘が残っていれば件数を添えて失敗する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			root := writeActionRoot(t, quotedDefectAction)

			err := run(t.Context(), stubWD(root), stubLookPath(nil))

			require.ErrorIs(t, err, errFindings)
			assert.ErrorContains(t, err, "検査 1 ステップ")
		})
	})
}
