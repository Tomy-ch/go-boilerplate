package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// subtestGroupSelfFile は、本ファイル自身の名前です。走査対象から外すために使います。
// 合成入力として書いた違反を自己走査で拾わないよう除外します。
const subtestGroupSelfFile = "subtest_group_policy_test.go"

var (
	// testFuncDeclRe は、レシーバを持たないトップレベル関数の宣言を検出する。
	// テスト関数以外にもマッチするのは意図的で、ヘルパー関数へ入った時点で走査を止めるために名前を捕捉する。
	testFuncDeclRe = regexp.MustCompile(`^func (\w+)\(`)
	// wrappedRunRe は、引数が同一行に無い t.Run 呼び出しを検出する。
	// ケース名を次行へ折り返すとインデント段数による構造判定が成立しないため、解析不能として違反にする（黙って読み飛ばすと検出漏れになる）。
	wrappedRunRe = regexp.MustCompile(`^\t+t\.Run\($`)
	// topLevelRunRe は、TestXxx 直下（インデント 1 段）の t.Run を検出する。
	topLevelRunRe = regexp.MustCompile(`^\tt\.Run\(`)
	// literalGroupRe は、最外周グループとして許容される唯一の形にマッチする。
	literalGroupRe = regexp.MustCompile(`^\tt\.Run\("(正常系|異常系)", func\(`)
	// nestedGroupRe は、グループの内側（インデント 2 段以上）でサブケース名が 正常系 / 異常系 で始まるものを検出する。
	// 規約はグループ内でのさらなるネストを許すため、深さを 2 段に固定すると 3 段目以降のプレフィックス形式を取りこぼす。
	nestedGroupRe = regexp.MustCompile(`^\t{2,}t\.Run\("(正常系|異常系)`)
)

// TestSubtestGroupPolicy は、テストのサブテスト構造が規約どおりであることを機械検証する。
// t.Run サブケースを持つ TestXxx の最外周はリテラル 正常系 / 異常系 に限り、
// 挙動文の直置き・正常系_xxx のプレフィックス形式・境界ケースのような第 3 のグループはいずれも違反になる。
// 単一シナリオの TestXxx はグループを省略してよいため、t.Run を 1 つも持たない関数は対象外。
//
// gofmt 済みであることを前提に、インデント段数でネスト深さを判定する。
// どちら側のグループへ属すべきかは意味の判断であり機械検証できないため、構造のみを対象とし、
// 帰属は testing-conventions.md §10 とレビューが担う。構造違反を許す allowlist は持たない。
func TestSubtestGroupPolicy(t *testing.T) {
	t.Parallel()

	var violations []string
	scanned := 0

	for _, root := range moduleSubdirs(t, "internal", "pkg") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") || filepath.Base(path) == subtestGroupSelfFile {
				return nil
			}
			src, rerr := pkgfs.OS{}.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			scanned++
			violations = append(violations, collectSubtestGroupViolations(path, strings.Split(string(src), "\n"))...)
			return nil
		})
		require.NoError(t, err)
	}

	require.NotZero(t, scanned, "走査対象のテストファイルが 0 件（moduleSubdirs のルート解決を疑う）")

	sort.Strings(violations)
	for _, v := range violations {
		t.Log("サブテスト構造の違反: " + v)
	}

	require.Empty(t, violations,
		"サブテストのグループ構造が規約に反している。t.Run を使う TestXxx の最外周はリテラル 正常系 / 異常系 に限り、"+
			"分割の余地がない単一シナリオはグループごと省略すること。")
}

// collectSubtestGroupViolations は、規約に反するサブテストのグループ構造を違反として列挙する。
func collectSubtestGroupViolations(file string, lines []string) []string {
	var violations []string
	inTestFunc := false

	for i, line := range lines {
		if m := testFuncDeclRe.FindStringSubmatch(line); m != nil {
			inTestFunc = strings.HasPrefix(m[1], "Test")
			continue
		}
		if !inTestFunc {
			continue
		}

		switch {
		case wrappedRunRe.MatchString(line):
			violations = append(violations, describe(file, i, line,
				"t.Run の引数が同一行に無く構造を解析できない"))
		case topLevelRunRe.MatchString(line) && !literalGroupRe.MatchString(line):
			violations = append(violations, describe(file, i, line,
				"TestXxx 直下の t.Run はリテラル 正常系 / 異常系 のみ"))
		case nestedGroupRe.MatchString(line):
			violations = append(violations, describe(file, i, line,
				"グループ内のサブケース名に 正常系 / 異常系 を含めない"))
		}
	}
	return violations
}

func Test_collectSubtestGroupViolations(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リテラルの正常系グループは違反にしない", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"正常系\", func(t *testing.T) {",
				"\t\tt.Run(\"入力が正しい場合、値を返す\", func(t *testing.T) {",
			}

			assert.Empty(t, collectSubtestGroupViolations("f_test.go", lines))
		})

		t.Run("t.Run を持たない単一シナリオの TestXxx は違反にしない", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Parallel()",
				"\tassert.NotNil(t, Foo())",
			}

			assert.Empty(t, collectSubtestGroupViolations("f_test.go", lines))
		})

		t.Run("テスト関数の外にある同形の行は走査しない", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func helperRun(t *testing.T) {",
				"\tt.Run(\"挙動を説明する文\", func(t *testing.T) {",
			}

			assert.Empty(t, collectSubtestGroupViolations("f_test.go", lines))
		})

		t.Run("コメントアウトされた t.Run は走査しない", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\t// t.Run(\"挙動を説明する文\", func(t *testing.T) {",
			}

			assert.Empty(t, collectSubtestGroupViolations("f_test.go", lines))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("挙動文を直下に置く形を違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"入力が正しい場合、値を返す\", func(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "リテラル 正常系 / 異常系 のみ")
		})

		t.Run("正常系_xxx のプレフィックス形式を違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"正常系: 入力が正しい場合、値を返す\", func(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "リテラル 正常系 / 異常系 のみ")
		})

		t.Run("境界ケースのような第3のグループを違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"境界ケース\", func(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "リテラル 正常系 / 異常系 のみ")
		})

		t.Run("グループ内のサブケース名に正常系を含む形を違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"正常系\", func(t *testing.T) {",
				"\t\tt.Run(\"正常系_入力が正しい場合、値を返す\", func(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "サブケース名に 正常系 / 異常系 を含めない")
		})

		t.Run("さらにネストした3段目のプレフィックス形式も違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"正常系\", func(t *testing.T) {",
				"\t\tt.Run(\"さらに分類する\", func(t *testing.T) {",
				"\t\t\tt.Run(\"正常系_入力が正しい場合、値を返す\", func(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "サブケース名に 正常系 / 異常系 を含めない")
		})

		t.Run("ケース名を次行へ折り返した t.Run を解析不能として違反にする", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"正常系\", func(t *testing.T) {",
				"\t\tt.Run(",
				"\t\t\t\"正常系_折り返しで隠したプレフィックス形式\",",
				"\t\t\tfunc(t *testing.T) {",
			}

			got := collectSubtestGroupViolations("f_test.go", lines)

			require.Len(t, got, 1)
			assert.Contains(t, got[0], "同一行に無く構造を解析できない")
		})

		t.Run("複数の違反をすべて列挙する", func(t *testing.T) {
			t.Parallel()

			lines := []string{
				"func TestFoo(t *testing.T) {",
				"\tt.Run(\"挙動を説明する文\", func(t *testing.T) {",
				"func TestBar(t *testing.T) {",
				"\tt.Run(\"境界ケース\", func(t *testing.T) {",
			}

			assert.Len(t, collectSubtestGroupViolations("f_test.go", lines), 2)
		})
	})
}
