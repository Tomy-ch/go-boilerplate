package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipReasonMaxLines は、1 つの Skip 呼び出しを追う行数の上限です（gofmt の折り返しを吸収する）。
// 上限内に閉じ括弧が現れない呼び出しは「解析できない」として違反にします（黙って読み飛ばすと検出漏れになる）。
const skipReasonMaxLines = 8

// selfFile は、本ファイル自身の名前です。走査対象から外すために使います。
// 合成入力として書いた違反を自己走査で拾わないよう除外します。
const selfFile = "skip_reason_policy_test.go"

var (
	// skipCallRe は、テストハンドル経由の Skip 呼び出し（t.Skip / tb.Skipf / t.SkipNow 等）を検出する。
	// SkipNow を含めるのは、理由を持たない skip がゲートを素通りしないようにするため。
	skipCallRe = regexp.MustCompile(`\b\w+\.Skip(f|Now)?\(`)
	// skipNowRe は、理由を伴わない SkipNow 呼び出しを判別する。
	skipNowRe = regexp.MustCompile(`\b\w+\.SkipNow\(`)
	// coveringTestRe は、skip 理由が別のテスト関数を名指ししていることを検出する。
	// Go のテスト関数は Test の直後が大文字か `_` になるため、この形で「他テスト参照」を捕捉できる。
	coveringTestRe = regexp.MustCompile(`Test[A-Z_]`)
)

// TestSkipReasonDoesNotNameCoveringTest は、Skip の理由が他のテストを名指ししていないことを機械検証する。
// 「他のテストでカバー済み」が無効な skip 理由である根拠は docs/testing-conventions.md §1 が定める。
//
// 理由は文字列リテラル直書きに限る。変数越しに渡されると本文をテキストで追えず、
// 一段挟むだけで規約を骨抜きにできてしまうため、リテラル以外は違反として扱う。
func TestSkipReasonDoesNotNameCoveringTest(t *testing.T) {
	t.Parallel()

	var violations []string
	for _, root := range moduleSubdirs(t, "internal", "pkg") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") || filepath.Base(path) == selfFile {
				return nil
			}
			src, rerr := pkgfs.OS{}.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			violations = append(violations, collectSkipReasonViolations(path, strings.Split(string(src), "\n"))...)
			return nil
		})
		require.NoError(t, err)
	}

	sort.Strings(violations)
	for _, v := range violations {
		t.Log("skip 理由の違反: " + v)
	}

	require.Empty(t, violations,
		"skip の理由が規約に反している。テスト可能な対象は実テストを書くこと。"+
			"検証不可能であるために到達できない場合に限り、なぜ検証不可能かを文字列リテラルで書いて skip できる。")
}

// collectSkipReasonViolations は、規約に反する Skip 呼び出しを違反として列挙する。
func collectSkipReasonViolations(file string, lines []string) []string {
	var violations []string
	for i, line := range lines {
		loc := skipCallRe.FindStringIndex(line)
		if loc == nil || inComment(line, loc[0]) {
			continue
		}

		if skipNowRe.MatchString(line[loc[0]:loc[1]]) {
			violations = append(violations, describe(file, i, lines[i], "理由を持たない SkipNow は使えない"))
			continue
		}

		reason, ok := skipCallText(lines, i, loc[1])
		// 折り返された呼び出しはリテラルが次行以降に来るため、連結後の先頭空白を落としてから判定する。
		literal := strings.TrimLeft(reason, " \t")
		switch {
		case !ok:
			violations = append(violations, describe(file, i, lines[i], "理由を解析できない（複数行が長すぎる）"))
		case !strings.HasPrefix(literal, `"`) && !strings.HasPrefix(literal, "`"):
			violations = append(violations, describe(file, i, lines[i], "理由が文字列リテラルではない"))
		case coveringTestRe.MatchString(reason):
			violations = append(violations, describe(file, i, lines[i], "理由が他のテストを名指ししている"))
		}
	}
	return violations
}

// describe は、違反 1 件の報告文字列を組み立てる。
func describe(file string, idx int, line, reason string) string {
	return file + ":" + strconv.Itoa(idx+1) + ": " + reason + ": " + strings.TrimSpace(line)
}

// inComment は、位置 at がその行のコメント部分にあるかを返す。
// 行頭コメントだけでなく、コード末尾に付いた行内コメント中の言及も走査対象から外すために使う。
func inComment(line string, at int) bool {
	c := commentStart(line)
	return c >= 0 && c < at
}

// commentStart は、行コメントの開始位置を返す（無ければ -1）。
// 文字列リテラル中の `//`（URL など）をコメント開始と誤認すると、同じ行にある実在の Skip 呼び出しを
// 走査対象から外してしまうため、クオートの開閉を追いながらリテラルの外だけを探す。
func commentStart(line string) int {
	var inQuote, inRaw, escaped bool
	for i := range len(line) {
		switch {
		case escaped:
			escaped = false
		case inRaw:
			inRaw = line[i] != '`'
		case inQuote && line[i] == '\\':
			escaped = true
		case inQuote:
			inQuote = line[i] != '"'
		case line[i] == '`':
			inRaw = true
		case line[i] == '"':
			inQuote = true
		case line[i] == '/' && i+1 < len(line) && line[i+1] == '/':
			return i
		}
	}
	return -1
}

// skipCallText は、Skip 呼び出しの引数テキストを返す。
// gofmt は長い引数を複数行へ折り返すため、閉じ括弧で終わる行までを連結する。
// 上限行数内に閉じ括弧が見つからない場合は ok=false を返し、呼び出し元が違反として扱う。
func skipCallText(lines []string, declIdx, from int) (string, bool) {
	var text strings.Builder
	text.WriteString(lines[declIdx][from:])
	if strings.HasSuffix(strings.TrimSpace(lines[declIdx]), ")") {
		return text.String(), true
	}
	for j := declIdx + 1; j < len(lines) && j-declIdx <= skipReasonMaxLines; j++ {
		text.WriteString(lines[j])
		if strings.HasSuffix(strings.TrimSpace(lines[j]), ")") {
			return text.String(), true
		}
	}
	return text.String(), false
}

func Test_collectSkipReasonViolations(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検証不可能な理由を述べた skip は違反にしない", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.Skip("tb.Fatalf は呼び出し側テストの終了を伴うため検証不可")`}

			assert.Empty(t, collectSkipReasonViolations("f.go", lines))
		})

		t.Run("行頭コメント中の記述は走査しない", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	// 旧実装では t.Skip("TestFoo でカバー") と書いていた`}

			assert.Empty(t, collectSkipReasonViolations("f.go", lines))
		})

		t.Run("コード末尾の行内コメント中の記述は走査しない", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	doSomething() // かつては t.Skip("TestFoo 参照") だった`}

			assert.Empty(t, collectSkipReasonViolations("f.go", lines))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("理由が他のテストを名指ししていれば違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.Skip("TestCoveringOne でカバー済み")`}

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "他のテストを名指し")
		})

		t.Run("同じ行の先行リテラルに含まれる // で見逃さない", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	u := "http://internal"; t.Skip("TestCoveringOne でカバー済み")`}

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "他のテストを名指し")
		})

		t.Run("理由を持たない SkipNow を違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.SkipNow()`}

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "SkipNow")
		})

		t.Run("理由を変数越しに渡す回避を違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.Skip(reason)`}

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "文字列リテラルではない")
		})

		t.Run("複数行に折り返した理由の末尾にある名指しも検出する", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				`	t.Skip(`,
				`		"前半は検証不可能性の説明で" +`,
				`			"末尾で TestCoveringOne を名指しする")`,
			}

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "他のテストを名指し")
		})

		t.Run("上限行数内に閉じ括弧が現れない呼び出しは解析不能として違反にする", func(t *testing.T) {
			t.Parallel()

			lines := make([]string, 0, skipReasonMaxLines+3)
			lines = append(lines, `	t.Skip(`)
			for range skipReasonMaxLines + 1 {
				lines = append(lines, `		"継続行" +`)
			}
			lines = append(lines, `		"末尾で TestCoveringOne を名指しする")`)

			got := collectSkipReasonViolations("f.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "解析できない")
		})
	})
}

func Test_skipCallText(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一行の呼び出しは引数テキストをそのまま返す", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.Skip("理由")`}

			got, ok := skipCallText(lines, 0, len(`	t.Skip(`))

			require.True(t, ok)
			assert.Equal(t, `"理由")`, got)
		})

		t.Run("閉じ括弧の行まで連結する", func(t *testing.T) {
			t.Parallel()

			lines := []string{`	t.Skip(`, `		"前半" +`, `			"後半")`, `	assert.True(t, ok)`}

			got, ok := skipCallText(lines, 0, len(`	t.Skip(`))

			require.True(t, ok)
			assert.Contains(t, got, "前半")
			assert.Contains(t, got, "後半")
			assert.NotContains(t, got, "assert.True") // 呼び出しの外まで読み進めない
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限行数内に閉じ括弧が無ければ ok=false を返す", func(t *testing.T) {
			t.Parallel()

			lines := make([]string, 0, skipReasonMaxLines+2)
			lines = append(lines, `	t.Skip(`)
			for range skipReasonMaxLines + 1 {
				lines = append(lines, `		"継続行" +`)
			}

			_, ok := skipCallText(lines, 0, len(`	t.Skip(`))

			assert.False(t, ok)
		})
	})
}

func Test_commentStart(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行頭コメントの開始位置を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 1, commentStart(` // 理由`))
		})

		t.Run("コード末尾の行内コメントの開始位置を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, len(`x := 1 `), commentStart(`x := 1 // 理由`))
		})

		t.Run("コメントが無ければ -1 を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, -1, commentStart(`t.Skip("理由")`))
		})

		t.Run("文字列リテラル中の // はコメント開始とみなさない", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, -1, commentStart(`u := "http://internal"`))
		})

		t.Run("生文字列リテラル中の // もコメント開始とみなさない", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, -1, commentStart("u := `http://internal`"))
		})

		t.Run("エスケープされたクオートでリテラルは閉じない", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, -1, commentStart(`u := "a\"http://internal"`))
		})

		t.Run("リテラルを閉じた後の // はコメント開始とみなす", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, len(`u := "http://internal" `), commentStart(`u := "http://internal" // 理由`))
		})
	})
}
