package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/require"
)

// 命名規約: メソッドは `Test<Type>_<Method>`（Type が非公開なら `Test_<type>_<Method>`）、
// 自由関数は `Test<Func>`（非公開なら `Test_<func>`）。depguard が go/ast を禁じるため、
// gofmt 済みソースのテキスト走査で検出する（既存 architest と同方針）。
var (
	// 型パラメータ節（`[T any]` 等）を許容する。レシーバ型・自由関数名の直後に挟まりうるため、
	// 省略可能な `(?:\[.*\])?` を挟んでからメソッド引数の `(` を要求する。
	methodDeclRe = regexp.MustCompile(`^func \(\s*\w+\s+\*?(\w+)(?:\[.*\])?\) (\w+)\(`)
	freeFuncRe   = regexp.MustCompile(`^func (\w+)(?:\[.*\])?\(`)
	testFuncRe   = regexp.MustCompile(`^func (Test\w+)\(`)
	branchRe     = regexp.MustCompile(`^\t+(if|for|switch|select)[ ({]`)
	genHeaderRe  = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)
)

type prodSubject struct {
	name       string   // func / method 名（報告用）
	file       string   // 宣言元
	candidates []string // 許容するテスト関数名（いずれか 1 つでも存在すれば充足）
}

// TestUnitTestMappingCompleteness は、分岐（if/for/switch/select）を持つ production の
// func / method が、命名規約に一致する TestXxx を同一パッケージに持つことを機械検証する。
// テスト対象と TestXxx の 1:1 を強制し、レビュー頼みの見落とし（例: keyFunc の防御分岐が
// 呼び出し側テストに埋もれて専用テストが無い状態）を loud な失敗に変える。
//
// 「1:1 を採らない」ことは許さない。真にユニットテスト不要 / 現時点で書けない対象も、
// 命名規約どおりの TestXxx を宣言し中で t.Skip("理由") することで意図を明示する
// （allowlist は持たない。理由がコード上に残る）。
func TestUnitTestMappingCompleteness(t *testing.T) {
	t.Parallel()

	prodByDir, testsByDir := scanPackages(t)
	violations := collectViolations(prodByDir, testsByDir)
	sort.Strings(violations)
	for _, v := range violations {
		t.Log("1:1 未対応: " + v)
	}
	t.Logf("total 1:1 violations: %d", len(violations))

	require.Empty(t, violations,
		"分岐を持つ production 関数/メソッドに対応する TestXxx が無い（1:1 違反）。"+
			"専用テストを追加するか、真に不要なら命名規約どおりの TestXxx を宣言し t.Skip(理由) で明示すること。")
}

// scanPackages は internal / pkg 配下を走査し、ディレクトリ単位で
// production subject と TestXxx 名を収集して返す。
func scanPackages(t *testing.T) (map[string][]prodSubject, map[string]map[string]struct{}) {
	t.Helper()
	prodByDir := map[string][]prodSubject{}
	testsByDir := map[string]map[string]struct{}{}

	for _, root := range moduleSubdirs(t, "internal", "pkg") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			src, rerr := pkgfs.OS{}.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			indexGoFile(path, strings.Split(string(src), "\n"), prodByDir, testsByDir)
			return nil
		})
		require.NoError(t, err)
	}
	return prodByDir, testsByDir
}

// indexGoFile は 1 ファイルを分類する。_test.go は TestXxx 名を、production は分岐持ち subject を収集する。
func indexGoFile(path string, lines []string, prodByDir map[string][]prodSubject, testsByDir map[string]map[string]struct{}) {
	dir := filepath.Dir(path)
	if strings.HasSuffix(path, "_test.go") {
		if testsByDir[dir] == nil {
			testsByDir[dir] = map[string]struct{}{}
		}
		for _, l := range lines {
			if m := testFuncRe.FindStringSubmatch(l); m != nil {
				testsByDir[dir][m[1]] = struct{}{}
			}
		}
		return
	}
	if isGeneratedGo(path, lines) {
		return
	}
	prodByDir[dir] = append(prodByDir[dir], collectBranchingSubjects(lines, path)...)
}

// collectViolations は、対応する TestXxx を持たない subject を違反として列挙する。
func collectViolations(prodByDir map[string][]prodSubject, testsByDir map[string]map[string]struct{}) []string {
	var violations []string
	for dir, subs := range prodByDir {
		tests := testsByDir[dir]
		for _, s := range subs {
			if !hasAnyTest(tests, s.candidates) {
				violations = append(violations, s.file+": "+s.name+" → 期待テスト名: "+strings.Join(s.candidates, " / "))
			}
		}
	}
	return violations
}

// collectBranchingSubjects は、分岐を持つ func / method を subject 化して返す。
// 分岐が無い trivial（getter / 単純ラッパ等）は 1:1 の対象外として除外する。
func collectBranchingSubjects(lines []string, file string) []prodSubject {
	var subs []prodSubject
	for i, line := range lines {
		var name string
		var candidates []string

		if m := methodDeclRe.FindStringSubmatch(line); m != nil {
			typ, method := m[1], m[2]
			name = typ + "." + method
			candidates = []string{"Test" + typ + "_" + method, "Test_" + typ + "_" + method}
		} else if m := freeFuncRe.FindStringSubmatch(line); m != nil {
			fn := m[1]
			if fn == "main" || fn == "init" {
				continue
			}
			name = fn
			candidates = []string{"Test" + fn, "Test_" + fn}
		} else {
			continue
		}

		if funcBodyHasBranch(lines, i) {
			subs = append(subs, prodSubject{name: name, file: file, candidates: candidates})
		}
	}
	return subs
}

// funcBodyHasBranch は、宣言行 declIdx から次の行頭 `}`（トップレベルの閉じ括弧）までを本文とみなし、
// 文頭に分岐キーワードが現れるかを判定する。コメント行（`//`）は除外される（branchRe が行頭タブ+キーワード限定のため）。
//
// gofmt は短い body を 1 行に保つ（例: getter `func (u *User) City() string { return u.city }`）。
// 1 行関数は trivial とみなし分岐なし扱いにする（複数行本文の走査に入ると後続関数の分岐を誤検出するため）。
func funcBodyHasBranch(lines []string, declIdx int) bool {
	if strings.HasSuffix(strings.TrimRight(lines[declIdx], " \t"), "}") {
		return false // 1 行関数 = trivial
	}
	for j := declIdx + 1; j < len(lines); j++ {
		if lines[j] == "}" { // gofmt: トップレベル関数の閉じ括弧は行頭 `}`
			return false
		}
		if branchRe.MatchString(lines[j]) {
			return true
		}
	}
	return false
}

func hasAnyTest(tests map[string]struct{}, candidates []string) bool {
	for _, c := range candidates {
		if _, ok := tests[c]; ok {
			return true
		}
	}
	return false
}

// isGeneratedGo は、生成物（ファイル名 or `// Code generated ... DO NOT EDIT.` ヘッダ）を判定する。
func isGeneratedGo(path string, lines []string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".gen.go") || strings.HasSuffix(base, ".sql.go") ||
		strings.HasSuffix(base, "_mock.go") || strings.HasSuffix(base, ".pb.go") {
		return true
	}
	for k := 0; k < len(lines) && k < 5; k++ {
		if genHeaderRe.MatchString(lines[k]) {
			return true
		}
	}
	return false
}

// moduleSubdirs は、本テストファイル位置（internal/architest）からモジュールルートを辿り、
// 指定サブディレクトリの絶対パス群を返す。
func moduleSubdirs(t *testing.T, subs ...string) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..") // internal/architest -> module root
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		out = append(out, filepath.Join(root, s))
	}
	return out
}
