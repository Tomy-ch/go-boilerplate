package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
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
	// レシーバ変数名は省略可能（`func (*T) M()` は Go として合法で、名前を省くだけで検証を逃れられないようにする）。
	methodDeclRe = regexp.MustCompile(`^func \(\s*(?:\w+\s+)?\*?(\w+)(?:\[.*\])?\) (\w+)\(`)
	freeFuncRe   = regexp.MustCompile(`^func (\w+)(?:\[.*\])?\(`)
	testFuncRe   = regexp.MustCompile(`^func (Test\w+)\(`)
	genHeaderRe  = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)
)

type prodSubject struct {
	name       string   // func / method 名（報告用）
	file       string   // 宣言元
	candidates []string // 許容するテスト関数名（いずれか 1 つでも存在すれば充足）
}

// TestUnitTestMappingCompleteness は、production の全 func / method が、命名規約に一致する
// TestXxx を同一パッケージに持つことを機械検証する。テスト対象と TestXxx の 1:1 を強制し、
// レビュー頼みの見落とし（例: keyFunc の防御分岐が呼び出し側テストに埋もれて専用テストが
// 無い状態、ルート登録だけを行う分岐なしの BindHandler が未テストのまま残る状態）を
// loud な失敗に変える。
//
// 「1:1 を採らない」ことは許さない。対象が検証不可能であるために到達できない場合に限り、
// 命名規約どおりの TestXxx を宣言し中で t.Skip("なぜ検証不可能か") することで意図を明示する
// （allowlist は持たない。理由がコード上に残る）。「他のテストでカバー済み」は skip の
// 理由にならない。テスト可能な対象は実テストを書く。
//
// 候補名が複数あるのは非公開シンボルの命名揺れを許容するためであって、両方を並べてよい
// という意味ではないため、候補名の TestXxx を 2 つ以上持つ subject も違反とする。
// 逆方向の検査はここまでで、候補名に一致しない TestXxx が同じ subject を対象にしている
// 状態（1 シンボルが規約外の名前のテストと分業している状態）は検知できない。テスト名から
// subject を逆引きする手段が無く、production に対応シンボルを持たない TestXxx は
// docs/testing-conventions.md §1 が by construction 正当と認めているため、そこまで踏み込むと
// 検出のほとんどが誤検知になる。その形は test-review のレビューが受け持つ。
func TestUnitTestMappingCompleteness(t *testing.T) {
	t.Parallel()

	prodByDir, testsByDir := scanPackages(t)
	violations := collectViolations(prodByDir, testsByDir)
	sort.Strings(violations)
	for _, v := range violations {
		t.Log("1:1 違反: " + v)
	}
	t.Logf("total 1:1 violations: %d", len(violations))

	require.Empty(t, violations,
		"production 関数/メソッドと TestXxx の 1:1 が崩れている。"+
			"対応する TestXxx が無い場合は専用テストを追加すること。検証不可能であるために到達できない"+
			"対象に限り、命名規約どおりの TestXxx を宣言し、なぜ検証不可能かを t.Skip の理由文に書いて"+
			"明示できる。候補名の TestXxx が重複している場合は、どちらか 1 本へケースを統合すること。")
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
	prodByDir[dir] = append(prodByDir[dir], collectSubjects(lines, path)...)
}

// collectViolations は、対応する TestXxx を持たない subject と、候補名の TestXxx を
// 2 つ以上持つ subject を違反として列挙する。
func collectViolations(prodByDir map[string][]prodSubject, testsByDir map[string]map[string]struct{}) []string {
	var violations []string
	for dir, subs := range prodByDir {
		tests := testsByDir[dir]
		for _, s := range subs {
			switch matched := matchingTests(tests, s.candidates); len(matched) {
			case 0:
				violations = append(violations, s.file+": "+s.name+" → 期待テスト名: "+strings.Join(s.candidates, " / "))
			case 1:
			default:
				violations = append(violations, s.file+": "+s.name+" → 候補名の TestXxx が重複: "+strings.Join(matched, " / "))
			}
		}
	}
	return violations
}

// collectSubjects は、宣言されている全ての func / method を subject 化して返す。
// 分岐の有無・body の行数では絞らない（getter / 単純ラッパ / ルート登録のみの関数も
// 契約を持ちうるため）。除外するのは main / init のみ。
func collectSubjects(lines []string, file string) []prodSubject {
	var subs []prodSubject
	for _, line := range lines {
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

		subs = append(subs, prodSubject{name: name, file: file, candidates: candidates})
	}
	return subs
}

// matchingTests は、candidates のうち実在する TestXxx 名を宣言順で返す。
func matchingTests(tests map[string]struct{}, candidates []string) []string {
	var matched []string
	for _, c := range candidates {
		if _, ok := tests[c]; ok {
			matched = append(matched, c)
		}
	}
	return matched
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

// moduleSubdirs は、モジュールルート配下の指定サブディレクトリの絶対パス群を返す。
func moduleSubdirs(t *testing.T, subs ...string) []string {
	t.Helper()
	root := moduleRoot(t)
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		out = append(out, filepath.Join(root, s))
	}
	return out
}
