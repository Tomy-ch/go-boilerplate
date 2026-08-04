package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func testFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for path, body := range files {
		fsys[path] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

// requireShellcheck は実物の shellcheck を要求する。手元に無い環境では skip するが、skip は既定の
// 出力に現れないため、CI のように検査されたことを保証したい実行では REQUIRE_SHELLCHECK を立てて
// 落とす。緑と「検査していない」が見分けられない状態こそ、このツール自身が塞いでいる欠陥にあたる。
func requireShellcheck(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(shellcheckBin); err == nil {
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

func TestParseAction(t *testing.T) {
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
	})
}

func TestActionFiles(t *testing.T) {
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
	})
}

func TestCollectSteps(t *testing.T) {
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
	})
}

func TestCountRunSteps(t *testing.T) {
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
	})
}

func TestBlockIndentWidth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空行を読み飛ばして最初の非空行のインデント幅を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 8, blockIndentWidth([]byte("run: |\n\n        echo hi\n"), 2))
		})

		t.Run("本文が空行だけなら 0 を返す", func(t *testing.T) {
			t.Parallel()
			assert.Zero(t, blockIndentWidth([]byte("run: |\n\n\n"), 2))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本文開始行が行範囲の外なら 0 を返す", func(t *testing.T) {
			t.Parallel()
			data := []byte("run: |\n        echo hi\n")
			assert.Zero(t, blockIndentWidth(data, 0))
			assert.Zero(t, blockIndentWidth(data, 99))
		})
	})
}

func TestShellDialect(t *testing.T) {
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

func TestMaskExpressions(t *testing.T) {
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
	})
}

func TestExprEnd(t *testing.T) {
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
	})
}

func TestRemapFindings(t *testing.T) {
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

		t.Run("解析できない行は無視する", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, remapFindings(step{file: "action.yaml", firstLine: 1}, "unexpected output\n"))
		})
	})
}

func TestRunShellcheck(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問題のない本文では指摘を返さない", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			out, err := runShellcheck(t.Context(), shebangs["bash"], "echo \"${HOME}\"\n")
			require.NoError(t, err)
			assert.Empty(t, strings.TrimSpace(out))
		})

		t.Run("クォート漏れを SC2086 として検出する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			out, err := runShellcheck(t.Context(), shebangs["bash"], "x=\"a b\"\necho $x\n")
			require.NoError(t, err)
			assert.Contains(t, out, "SC2086")
		})

		t.Run("shebang 行を除いた本文の行番号で報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			out, err := runShellcheck(t.Context(), shebangs["bash"], "x=\"a b\"\necho $x\n")
			require.NoError(t, err)
			assert.Contains(t, out, "-:3:")
		})

		t.Run("sh 方言では bash 専用構文を指摘する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			out, err := runShellcheck(t.Context(), shebangs["sh"], "echo \"${BASH_SOURCE[0]}\"\n")
			require.NoError(t, err)
			assert.Contains(t, out, "[SC3028]")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指摘以外の失敗は握り潰さずエラーにする", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)
			_, err := runShellcheck(canceledContext(t), shebangs["bash"], "echo ok\n")
			require.Error(t, err)
			require.ErrorIs(t, err, errShellcheck)
		})
	})
}

func TestCheck(t *testing.T) {
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
			require.ErrorIs(t, err, errShellcheck)
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
