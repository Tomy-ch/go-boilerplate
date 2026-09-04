package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// today は判定の基準日。期限の前後を跨ぐ値を固定して、実行日に依存しないようにする。
var today = time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)

// writeFile は root からの相対パスへ書き、途中のディレクトリを作る。
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()

	full := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

// lockWith は指定した解決キーを持つ lockfile の中身を組み立てる。キーは実物の書式のまま渡す。
func lockWith(keys ...string) string {
	var b strings.Builder

	b.WriteString("lockfileVersion: '9.0'\n\npackages:\n")

	for _, k := range keys {
		b.WriteString("  " + k + ":\n    resolution: {integrity: sha512-x}\n")
	}

	return b.String()
}

// setup は workspace / lockfile / バイパスの 3 点を書いた一時 root を返す。
func setup(t *testing.T, workspace, lock, bypasses string) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, root, filepath.Join("app", workspaceFile), workspace)
	writeFile(t, root, filepath.Join("app", lockFile), lock)
	writeFile(t, root, bypassFile, bypasses)

	return root
}

// bypassLine は 1 エントリ分の行を組み立てる。
func bypassLine(spec, expires string) string {
	return `"` + spec + `" = { expires = ` + expires + `, issue = 1479, reason = "テストの理由" }` + "\n"
}

// tooLongLine は bufio.Scanner の上限を超える 1 行を返す。
func tooLongLine() string {
	return strings.Repeat("x", bufio.MaxScanTokenSize+1) + "\n"
}

// wantOneExclusion は 1 件の例外を読み出せたことを確かめる。書式ごとのケースが共有する終端。
func wantOneExclusion(t *testing.T, root string) {
	t.Helper()

	got, err := parseWorkspace(root, "app/"+workspaceFile)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "a@1", got[0].spec)
}

func Test_parseWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pnpm が honor するブロックシーケンスを読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude:\n  - a@1\n")

			wantOneExclusion(t, root)
		})

		t.Run("pnpm が honor するキーと同じ桁のシーケンスを読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude:\n- a@1\n")

			wantOneExclusion(t, root)
		})

		t.Run("pnpm が honor するフロー形式を読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude: [a@1]\n")

			wantOneExclusion(t, root)
		})

		t.Run("pnpm が honor するコロン前に空白のあるキーを読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude :\n  - a@1\n")

			wantOneExclusion(t, root)
		})

		t.Run("pnpm が honor する単一引用符つきの項目を読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude:\n  - 'a@1'\n")

			wantOneExclusion(t, root)
		})

		t.Run("pnpm が honor する二重引用符つきの項目を読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude:\n  - \"a@1\"\n")

			wantOneExclusion(t, root)
		})

		t.Run("空リストの明示はエントリゼロとして読む", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAgeExclude: []\n")

			got, err := parseWorkspace(root, "app/"+workspaceFile)

			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("キーが無ければエントリゼロとして読む", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "minimumReleaseAge: 10080\n")

			got, err := parseWorkspace(root, "app/"+workspaceFile)

			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("複数エントリを行番号付きで読み出す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile),
				"minimumReleaseAgeExclude:\n  - a@1\n  - b@2\n")

			got, err := parseWorkspace(root, "app/"+workspaceFile)

			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, 2, got[0].line)
			assert.Equal(t, 3, got[1].line)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言ファイルが読めない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := parseWorkspace(t.TempDir(), "missing/"+workspaceFile)

			require.Error(t, err)
		})

		t.Run("YAML として壊れている場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "a:\n- b\n  c: [\n")

			_, err := parseWorkspace(root, "app/"+workspaceFile)

			require.Error(t, err)
		})
	})
}

func Test_findExcludeSequence(t *testing.T) {
	t.Parallel()

	// node は YAML を 1 つのノード木へ読み、findExcludeSequence へ直接渡せる形にする。
	node := func(t *testing.T, src string) *yaml.Node {
		t.Helper()

		var doc yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(src), &doc))

		return &doc
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("シーケンスノードを中身ごと返す", func(t *testing.T) {
			t.Parallel()

			got := findExcludeSequence(node(t, "minimumReleaseAgeExclude:\n  - a@1\n  - b@2\n"))

			require.NotNil(t, got)
			require.Len(t, got.Content, 2)
			assert.Equal(t, "a@1", got.Content[0].Value)
			assert.Equal(t, "b@2", got.Content[1].Value)
		})

		t.Run("他のキーが並んでいても該当キーを選ぶ", func(t *testing.T) {
			t.Parallel()

			got := findExcludeSequence(node(t,
				"minimumReleaseAge: 10080\nminimumReleaseAgeExclude:\n  - a@1\nminimumReleaseAgeStrict: true\n"))

			require.NotNil(t, got)
			require.Len(t, got.Content, 1)
			assert.Equal(t, "a@1", got.Content[0].Value)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーが無ければ nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, findExcludeSequence(node(t, "minimumReleaseAge: 10080\n")))
		})

		t.Run("値がシーケンスでなければ nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, findExcludeSequence(node(t, "minimumReleaseAgeExclude: a@1\n")))
		})

		t.Run("トップレベルがマッピングでなければ nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, findExcludeSequence(node(t, "- a\n- b\n")))
		})

		t.Run("空の文書では nil を返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, findExcludeSequence(node(t, "")))
		})
	})
}

func Test_lockedSpecs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引用符と peer サフィックスを剥がして解決キーを集める", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, "l.yaml", lockWith(
				"acorn@8.18.0",
				"'@antfu/install-pkg@1.1.0'",
				"acorn-jsx@5.3.2(acorn@8.18.0)",
				`"quoted@1.0.0"`,
			))

			got, err := lockedSpecs(filepath.Join(root, "l.yaml"))

			require.NoError(t, err)
			assert.Contains(t, got, "acorn@8.18.0")
			assert.Contains(t, got, "@antfu/install-pkg@1.1.0", "scoped は quote されるので剥がす必要がある")
			assert.Contains(t, got, "acorn-jsx@5.3.2", "peer サフィックスを剥がさないと常に未解決に見える")
			assert.Contains(t, got, "quoted@1.0.0")
		})

		t.Run("@ を含まないキーは解決キーとして扱わない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, "l.yaml", lockWith("settings", "a@1"))

			got, err := lockedSpecs(filepath.Join(root, "l.yaml"))

			require.NoError(t, err)
			assert.NotContains(t, got, "settings")
			assert.Contains(t, got, "a@1")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読めない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := lockedSpecs(filepath.Join(t.TempDir(), "missing.yaml"))

			require.Error(t, err)
		})

		t.Run("行が長すぎて走査できない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, "l.yaml", tooLongLine())

			_, err := lockedSpecs(filepath.Join(root, "l.yaml"))

			require.Error(t, err, "走査の失敗を握り潰すと解決キーが空のまま残骸判定が狂う")
		})
	})
}

func Test_checkResolved(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解決済みなら空文字列", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), "")

			assert.Empty(t, checkResolved(root, exclusion{file: "app/" + workspaceFile, spec: "a@1"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解決していない版を違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), "")

			got := checkResolved(root, exclusion{file: "app/" + workspaceFile, spec: "b@2"})

			assert.Contains(t, got, "解決していません")
		})

		t.Run("lockfile が読めない場合も違反にする（fail-closed）", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "")

			got := checkResolved(root, exclusion{file: "app/" + workspaceFile, spec: "a@1"})

			assert.Contains(t, got, "読めません", "読めないことを解決済みへ倒すと lockfile を消すだけで検知を無効化できる")
		})
	})
}

func Test_parseBypasses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コメントと空行を読み飛ばしてエントリを読む", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, bypassFile, "# 見出し\n\n"+bypassLine("a@1", "2026-09-30"))

			got, err := parseBypasses(root)

			require.NoError(t, err)
			assert.Len(t, got, 1)
			assert.Equal(t, 1479, got["a@1"].issue)
			assert.Equal(t, "テストの理由", got["a@1"].reason)
			assert.Equal(t, 3, got["a@1"].line)
		})

		t.Run("ファイルが無ければ空とする", func(t *testing.T) {
			t.Parallel()

			got, err := parseBypasses(t.TempDir())

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できない行はエラーにする", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, bypassFile, `"a@1" = { expires = 2026-09-30 }`+"\n")

			_, err := parseBypasses(root)

			require.ErrorIs(t, err, errBypassInvalidLine, "読み飛ばすと書き損じが期限なしではなく無検査になる")
		})

		t.Run("キーの重複はエラーにする", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, bypassFile, bypassLine("a@1", "2026-09-30")+bypassLine("a@1", "2026-10-30"))

			_, err := parseBypasses(root)

			require.ErrorIs(t, err, errBypassDuplicateKey)
		})

		t.Run("不在以外の理由で開けない場合はエラーにする", func(t *testing.T) {
			t.Parallel()

			// .github をディレクトリではなくファイルにすると、その下を開く試みは ENOTDIR で失敗する。
			root := t.TempDir()
			writeFile(t, root, ".github", "")

			_, err := parseBypasses(root)

			require.Error(t, err, "読めない理由を一緒くたに空へ丸めると、例外ゼロと報告したまま素通りする")
			assert.NotErrorIs(t, err, os.ErrNotExist)
		})

		t.Run("行が長すぎて走査できない場合はエラーにする", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, bypassFile, tooLongLine())

			_, err := parseBypasses(root)

			require.Error(t, err)
		})
	})
}

func Test_parseBypassLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3 つのキーを読み出す", func(t *testing.T) {
			t.Parallel()

			got, key, err := parseBypassLine(strings.TrimSpace(bypassLine("a@1", "2026-09-30")), 9)

			require.NoError(t, err)
			assert.Equal(t, "a@1", key)
			assert.Equal(t, 9, got.line)
			assert.Equal(t, 1479, got.issue)
			assert.Equal(t, "テストの理由", got.reason)
			assert.Equal(t, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC), got.expires)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーが欠けていればエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseBypassLine(`"a@1" = { expires = 2026-09-30, issue = 1 }`, 9)

			require.ErrorIs(t, err, errBypassInvalidLine)
		})

		t.Run("読めない日付はエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseBypassLine(`"a@1" = { expires = 2026-02-30, issue = 1, reason = "x" }`, 9)

			require.Error(t, err)
		})

		t.Run("issue が数値範囲を超える場合はエラーにする", func(t *testing.T) {
			t.Parallel()

			_, _, err := parseBypassLine(
				`"a@1" = { expires = 2026-09-30, issue = 99999999999999999999999999, reason = "x" }`, 9,
			)

			require.Error(t, err, "正規表現は桁数を制限しないので、この経路は到達可能")
			assert.Contains(t, err.Error(), "の issue")
		})
	})
}

func Test_validate(t *testing.T) {
	t.Parallel()

	excl := []exclusion{{file: "app/" + workspaceFile, line: 9, spec: "a@1"}}

	// bypassesOf はバイパス lockfile を書いて読み直した結果を返す。
	bypassesOf := func(t *testing.T, root string) map[string]bypass {
		t.Helper()

		got, err := parseBypasses(root)
		require.NoError(t, err)

		return got
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内・解決済み・対応するバイパスがあれば違反ゼロ", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-09-30"))

			assert.Empty(t, validate(root, excl, bypassesOf(t, root), today))
		})

		t.Run("例外もバイパスも無ければ違反ゼロ", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, validate(t.TempDir(), nil, map[string]bypass{}, today))
		})

		t.Run("期限が当日なら未だ切れていない", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-09-05"))

			assert.Empty(t, validate(root, excl, bypassesOf(t, root), today))
		})

		t.Run("期限が上限ちょうどなら越えていない", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-12-05"))

			assert.Empty(t, validate(root, excl, bypassesOf(t, root), today))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限の無い例外を違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), "")

			got := validate(root, excl, map[string]bypass{}, today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "期限がありません")
		})

		t.Run("期限切れを違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-09-04"))

			got := validate(root, excl, bypassesOf(t, root), today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "切れています")
			assert.Contains(t, got[0], "#1479")
		})

		t.Run("上限の 3 ヶ月を越える期限を違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-12-06"))

			got := validate(root, excl, bypassesOf(t, root), today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "上限")
		})

		t.Run("どの例外にも対応しないバイパスを違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("z@9", "2026-09-30"))

			got := validate(root, nil, bypassesOf(t, root), today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "にもありません")
		})

		t.Run("期限切れと残骸を同時に報告する", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("other@1"), bypassLine("a@1", "2026-09-04"))

			got := validate(root, excl, bypassesOf(t, root), today)

			assert.Len(t, got, 2, "片方ずつ直させると往復が増える")
		})
	})
}

func Test_checkExclusion(t *testing.T) {
	t.Parallel()

	e := exclusion{file: "app/" + workspaceFile, line: 9, spec: "a@1"}
	limit := today.AddDate(0, maxBypassMonths, 0)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内かつ解決済みなら違反ゼロ", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), bypassLine("a@1", "2026-09-30"))
			bp, err := parseBypasses(root)
			require.NoError(t, err)

			assert.Empty(t, checkExclusion(root, e, bp, today, limit))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("バイパスが無ければ書式を添えて違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), "")

			got := checkExclusion(root, e, map[string]bypass{}, today, limit)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "expires = YYYY-MM-DD")
		})

		t.Run("期限は有効だが解決していない版は残骸だけを違反にする", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("other@1"), bypassLine("a@1", "2026-09-30"))
			bp, err := parseBypasses(root)
			require.NoError(t, err)

			got := checkExclusion(root, e, bp, today, limit)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "解決していません")
			assert.NotContains(t, got[0], "期限", "期限判定と解決判定が結合してはいけない")
		})
	})
}

func Test_findWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("配下の宣言ファイルを昇順で集める", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("b", workspaceFile), "")
			writeFile(t, root, filepath.Join("a", workspaceFile), "")

			got, err := findWorkspaces(root)

			require.NoError(t, err)
			assert.Equal(t, []string{"a/" + workspaceFile, "b/" + workspaceFile}, got)
		})

		t.Run("node_modules と vendor は他人の宣言なので除外する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeFile(t, root, filepath.Join("app", workspaceFile), "")
			writeFile(t, root, filepath.Join("app", "node_modules", "dep", workspaceFile), "")
			writeFile(t, root, filepath.Join("vendor", "dep", workspaceFile), "")

			got, err := findWorkspaces(root)

			require.NoError(t, err)
			assert.Equal(t, []string{"app/" + workspaceFile}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("走査できない場所ではエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := findWorkspaces(filepath.Join(t.TempDir(), "missing"))

			require.Error(t, err)
		})
	})
}

func Test_sortedKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("報告順を安定させるため昇順で返す", func(t *testing.T) {
			t.Parallel()

			got := sortedKeys(map[string]bypass{"b@2": {}, "a@1": {}})

			assert.Equal(t, []string{"a@1", "b@2"}, got)
		})
	})
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ nil を返し、件数を出力する", func(t *testing.T) {
			t.Parallel()

			root := setup(t,
				"minimumReleaseAgeExclude:\n  - a@1\n", lockWith("a@1"), bypassLine("a@1", "2026-09-30"))

			var out bytes.Buffer
			err := run(root, &out, today)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "例外 1 件 / バイパス 1 件")
			assert.Contains(t, out.String(), "違反なし")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フロー形式で書かれた期限の無い例外を検出する", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "minimumReleaseAgeExclude: [a@1]\n", lockWith("a@1"), "")

			var out bytes.Buffer
			err := run(root, &out, today)

			require.ErrorIs(t, err, errViolations, "行走査だとここが例外ゼロとして通っていた")
			assert.Contains(t, out.String(), "期限がありません")
		})

		t.Run("バイパスが壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "", lockWith("a@1"), "壊れた行\n")

			var out bytes.Buffer
			err := run(root, &out, today)

			require.ErrorIs(t, err, errBypassInvalidLine)
		})

		t.Run("走査できない root ではエラーを返す", func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			err := run(filepath.Join(t.TempDir(), "missing"), &out, today)

			require.Error(t, err, "握り潰すと workspace 0 件のまま違反なしを報告してしまう")
			assert.NotContains(t, out.String(), "違反なし")
		})

		t.Run("宣言ファイルが壊れていればエラーを返す", func(t *testing.T) {
			t.Parallel()

			root := setup(t, "a:\n- b\n  c: [\n", lockWith("a@1"), "")

			var out bytes.Buffer
			err := run(root, &out, today)

			require.Error(t, err, "握り潰すと隠れた例外が無視されたまま通る")
			assert.NotContains(t, out.String(), "違反なし")
		})
	})
}
