package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// today は判定の基準日。期限の前後を跨ぐ値を固定して、実行日に依存しないようにする。
var today = time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)

// workspaceWith は minimumReleaseAgeExclude ブロックを持つ最小の宣言ファイルを組み立てる。
func workspaceWith(entries ...string) string {
	var b strings.Builder
	b.WriteString("packages:\n  - \".\"\n\nminimumReleaseAge: 10080\n")
	if len(entries) == 0 {
		b.WriteString("minimumReleaseAgeExclude: []\n")
	} else {
		b.WriteString("minimumReleaseAgeExclude:\n")
		for _, e := range entries {
			b.WriteString("  - " + e + "\n")
		}
	}
	b.WriteString("minimumReleaseAgeStrict: true\n")

	return b.String()
}

// writeWorkspace は dir 配下に宣言ファイルと lockfile を書き、root からの相対パスを返す。
func writeWorkspace(t *testing.T, root, dir, content string, locked ...string) string {
	t.Helper()

	full := filepath.Join(root, dir)
	require.NoError(t, os.MkdirAll(full, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(full, workspaceFile), []byte(content), 0o600))

	var lock strings.Builder
	lock.WriteString("lockfileVersion: '9.0'\n\npackages:\n")
	for _, s := range locked {
		lock.WriteString("  " + s + ":\n    resolution: {integrity: sha512-x}\n")
	}
	require.NoError(t, os.WriteFile(filepath.Join(full, "pnpm-lock.yaml"), []byte(lock.String()), 0o600))

	return filepath.ToSlash(filepath.Join(dir, workspaceFile))
}

func Test_newExclusion(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("expires と issue と理由を読み出す", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "js-yaml@4.3.1",
				"expires: 2026-09-30 issue: 1479 GHSA-5p4m-2wfm-xmqj の修正版")

			assert.Empty(t, got.malformed)
			assert.Equal(t, "js-yaml", got.name)
			assert.Equal(t, "4.3.1", got.version)
			assert.Equal(t, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC), got.expires)
			assert.Equal(t, 1479, got.issue)
			assert.Equal(t, "GHSA-5p4m-2wfm-xmqj の修正版", got.reason)
		})

		t.Run("scope 付きパッケージの名前とバージョンを分ける", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "@scope/pkg@1.2.3",
				"expires: 2026-09-30 issue: 1 理由")

			assert.Empty(t, got.malformed)
			assert.Equal(t, "@scope/pkg", got.name)
			assert.Equal(t, "1.2.3", got.version)
		})

		t.Run("issue が # 付きでも読める", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "p@1", "expires: 2026-09-30 issue: #42 理由")

			assert.Empty(t, got.malformed)
			assert.Equal(t, 42, got.issue)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name@version の形でない場合は形式違反にする", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "js-yaml", "expires: 2026-09-30 issue: 1 理由")

			assert.Contains(t, got.malformed, "name@version")
		})

		t.Run("expires が無い場合は形式違反にする", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "p@1", "issue: 1 2026-09-30 以降に削除する")

			assert.Contains(t, got.malformed, "expires")
		})

		t.Run("issue が無い場合は形式違反にする", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "p@1", "expires: 2026-09-30 理由")

			assert.Contains(t, got.malformed, "issue")
		})

		t.Run("理由が無い場合は形式違反にする", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "p@1", "expires: 2026-09-30 issue: 1")

			assert.Contains(t, got.malformed, "理由")
		})

		t.Run("存在しない日付は形式違反にする", func(t *testing.T) {
			t.Parallel()

			got := newExclusion("a/pnpm-workspace.yaml", 9, "p@1", "expires: 2026-02-30 issue: 1 理由")

			assert.Contains(t, got.malformed, "expires")
			assert.True(t, got.expires.IsZero(), "読めない期限を有効な日付として持たない")
		})
	})
}

func Test_validate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeWorkspace(t, root, "app", workspaceWith(), "p@1.0.0")

	valid := func(expires string) exclusion {
		return newExclusion("app/pnpm-workspace.yaml", 9, "p@1.0.0", "expires: "+expires+" issue: 1 理由")
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内で lockfile が解決している例外は違反にならない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, validate(root, []exclusion{valid("2026-09-30")}, today))
		})

		t.Run("例外がゼロなら違反もゼロ", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, validate(root, nil, today))
		})

		t.Run("期限が当日なら未だ切れていない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, validate(root, []exclusion{valid("2026-09-05")}, today))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限切れを違反として報告する", func(t *testing.T) {
			t.Parallel()

			got := validate(root, []exclusion{valid("2026-09-04")}, today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "期限")
			assert.Contains(t, got[0], "切れています")
			assert.Contains(t, got[0], "#1")
		})

		t.Run("上限の 3 ヶ月を越える期限を違反として報告する", func(t *testing.T) {
			t.Parallel()

			got := validate(root, []exclusion{valid("2026-12-06")}, today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "上限")
		})

		t.Run("lockfile が解決していない例外を違反として報告する", func(t *testing.T) {
			t.Parallel()

			orphan := newExclusion("app/pnpm-workspace.yaml", 9, "p@9.9.9", "expires: 2026-09-30 issue: 1 理由")
			got := validate(root, []exclusion{orphan}, today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "解決していません")
		})

		t.Run("形式違反は期限より先に報告し、期限判定へ進めない", func(t *testing.T) {
			t.Parallel()

			broken := newExclusion("app/pnpm-workspace.yaml", 9, "p@1.0.0", "理由だけ")
			got := validate(root, []exclusion{broken}, today)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "expires")
			assert.NotContains(t, got[0], "切れています")
		})
	})
}

func Test_parseWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ブロック内のエントリを行番号付きで読み出す", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			rel := writeWorkspace(t, root, "app", workspaceWith(
				"a@1 # expires: 2026-09-30 issue: 1 理由",
				"b@2 # expires: 2026-09-30 issue: 2 理由",
			))

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, "a@1", got[0].spec)
			assert.Equal(t, "b@2", got[1].spec)
			assert.Equal(t, got[0].line+1, got[1].line)
		})

		t.Run("空リストの明示はエントリゼロとして読む", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			rel := writeWorkspace(t, root, "app", workspaceWith())

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("ブロックの後続キーをエントリとして拾わない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			rel := writeWorkspace(t, root, "app", workspaceWith("a@1 # expires: 2026-09-30 issue: 1 理由"))

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "a@1", got[0].spec)
		})

		t.Run("ブロック内のコメント行は読み飛ばす", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			content := "minimumReleaseAgeExclude:\n  # 補足\n  - a@1 # expires: 2026-09-30 issue: 1 理由\n"
			rel := writeWorkspace(t, root, "app", content)

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "a@1", got[0].spec)
		})

		t.Run("キーと同じ桁のブロックシーケンスも読み落とさない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			content := "minimumReleaseAgeExclude:\n- a@1 # expires: 2026-09-30 issue: 1 理由\n"
			rel := writeWorkspace(t, root, "app", content)

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 1, "インデントを条件にすると期限の無い例外が例外ゼロとして通る")
			assert.Equal(t, "a@1", got[0].spec)
		})

		t.Run("ブロック内の空行でブロックを終えない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			content := "minimumReleaseAgeExclude:\n  - a@1 # expires: 2026-09-30 issue: 1 理由\n\n  - b@2 # expires: 2026-09-30 issue: 2 理由\n"
			rel := writeWorkspace(t, root, "app", content)

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			assert.Len(t, got, 2)
		})

		t.Run("コメントの無いエントリも読み出して形式違反にする", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			rel := writeWorkspace(t, root, "app", workspaceWith("a@1"))

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.NotEmpty(t, got[0].malformed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リスト項目として読めない行を読み飛ばさず形式違反にする", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			content := "minimumReleaseAgeExclude:\n  -\n"
			rel := writeWorkspace(t, root, "app", content)

			got, err := parseWorkspace(root, rel)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.NotEmpty(t, got[0].malformed)
		})

		t.Run("宣言ファイルが読めない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := parseWorkspace(t.TempDir(), "missing/pnpm-workspace.yaml")

			require.Error(t, err)
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
			writeWorkspace(t, root, "b", workspaceWith())
			writeWorkspace(t, root, "a", workspaceWith())

			got, err := findWorkspaces(root)

			require.NoError(t, err)
			assert.Equal(t, []string{"a/pnpm-workspace.yaml", "b/pnpm-workspace.yaml"}, got)
		})

		t.Run("node_modules と vendor は他人の宣言なので除外する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeWorkspace(t, root, "app", workspaceWith())
			writeWorkspace(t, root, filepath.Join("app", "node_modules", "dep"), workspaceWith())
			writeWorkspace(t, root, filepath.Join("vendor", "dep"), workspaceWith())

			got, err := findWorkspaces(root)

			require.NoError(t, err)
			assert.Equal(t, []string{"app/pnpm-workspace.yaml"}, got)
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

func Test_resolvedInLock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lockfile が解決している版は true", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeWorkspace(t, root, "app", workspaceWith(), "p@1.0.0")

			assert.True(t, resolvedInLock(root, exclusion{file: "app/pnpm-workspace.yaml", spec: "p@1.0.0"}))
		})

		t.Run("lockfile が読めない場合は判定を諦めて true", func(t *testing.T) {
			t.Parallel()

			assert.True(t, resolvedInLock(t.TempDir(), exclusion{file: "none/pnpm-workspace.yaml", spec: "p@1.0.0"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("lockfile に無い版は false", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeWorkspace(t, root, "app", workspaceWith(), "p@1.0.0")

			assert.False(t, resolvedInLock(root, exclusion{file: "app/pnpm-workspace.yaml", spec: "p@9.9.9"}))
		})
	})
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反が無ければ nil を返し、件数を出力する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeWorkspace(t, root, "app",
				workspaceWith("p@1.0.0 # expires: 2026-09-30 issue: 1 理由"), "p@1.0.0")

			var out bytes.Buffer
			err := run(root, &out, today)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "workspace 1 件 / 例外 1 件")
			assert.Contains(t, out.String(), "違反なし")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("違反があればエラーを返し、内容を出力する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeWorkspace(t, root, "app",
				workspaceWith("p@1.0.0 # expires: 2026-09-04 issue: 1 理由"), "p@1.0.0")

			var out bytes.Buffer
			err := run(root, &out, today)

			require.Error(t, err)
			assert.Contains(t, out.String(), "切れています")
		})
	})
}
