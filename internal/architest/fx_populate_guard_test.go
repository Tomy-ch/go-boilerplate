package architest

import (
	"fmt"
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

const (
	fxPopulateCall = "fx.Populate("
	fxNewCall      = "fx.New("
)

var (
	// fxNewAssignRe は、fx.New の戻り値を受ける変数名（`app := fx.New(` / `app = fx.New(`）を取り出します。
	// ガードの受信者をこの変数に限定するため、構築とは無関係の `.Err()` をガードと誤認しません。
	fxNewAssignRe = regexp.MustCompile(`(\w+)\s*:?=\s*fx\.New\(`)
	// closureStartRe は、関数リテラルがブロックを開く行にマッチします。
	// gofmt は開き波括弧を行末に置くため、行末 `{` を条件にできます。
	closureStartRe = regexp.MustCompile(`func\s*\(.*\{$`)
	// identifierRe は、`&x` の x が Go の識別子ちょうどであることを検査します。
	identifierRe = regexp.MustCompile(`^[A-Za-z_]\w*$`)
)

// stripState は、raw string とブロックコメントが行をまたぐため、走査位置の文脈を持ち越します。
// 文字列リテラル（`"`）は行をまたがないので、行頭ごとに inQuote を落とします。
type stripState struct {
	inRaw   bool
	inBlock bool
	inQuote bool
}

// TestFxPopulateHasConstructionErrorGuard は、fx.Populate を使う production code の関数が、populate 対象を
// 参照する前に構築エラーを検査していることを機械検証する。
//
// fx.Populate は fx.New 時点の invoke であり、グラフ構築が失敗すると対象は nil のまま残る。
// そのまま参照すれば nil 参照で落ちる。fx.App.Start がエラーを短絡して返すため「Start を先に
// 呼んでいる限り安全」という状態は成立しうるが、それは実装順序という暗黙の前提にすぎず、
// コンパイラも lint も守らない。参照より前に構築エラーを検査する形を構造として固定し、
// 呼び出し箇所ごとに防御の有無が割れないようにする。
//
// 検出は depguard が go/ast を禁じるためテキスト走査で行う（既存 architest と同方針）。
// 走査前にコメントと raw string の中身を落とし、字面の一致で誤検出しないようにしている。
// 解析できるのは populate 対象が `&識別子` の直接形で、かつ fx.New の戻り値を変数で受けている
// 呼び出しに限る。それ以外（`fx.Annotate` 等で包んだ形）は対象を一意に特定できないため検査せず、
// 見送った件数をログに出してカバー範囲が黙って狭まらないようにする。
// _test.go はテストが構築失敗を注入して検証する正当な用法のため対象外。
func TestFxPopulateHasConstructionErrorGuard(t *testing.T) {
	t.Parallel()

	var violations []string
	sites, analyzed := 0, 0

	for _, root := range moduleSubdirs(t, "internal", "cmd") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := pkgfs.OS{}.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			lines := strings.Split(string(src), "\n")
			if isGeneratedGo(path, lines) {
				return nil
			}
			found, done, unguarded := collectUnguardedFxPopulate(lines, path)
			sites += found
			analyzed += done
			violations = append(violations, unguarded...)
			return nil
		})
		require.NoError(t, err)
	}

	// 走査対象が 0 件だと検証が常に成功してしまうため、ルート解決の破綻を空振りとして検出する。
	require.NotZero(t, analyzed, "解析できた fx.Populate が 0 件（moduleSubdirs のルート解決を疑う）")

	sort.Strings(violations)
	for _, v := range violations {
		t.Log("unguarded fx.Populate: " + v)
	}
	t.Logf("fx.Populate 呼び出し数: %d / 解析対象: %d / 解析対象外: %d / violations: %d",
		sites, analyzed, sites-analyzed, len(violations))

	require.Empty(t, violations,
		"fx.Populate の対象を構築エラーの検査より前に参照している。populate 対象に触れる前に "+
			"`if err := app.Err(); err != nil { ... }` で構築エラーを返すこと。")
}

// collectUnguardedFxPopulate は、gofmt 済みソースの行列から fx.Populate の呼び出し数・そのうち
// 解析できた数・populate 対象を構築エラーの検査より前に参照している箇所
// （`file:line: 該当行` 形式）を返します。
//
// 参照ごとに、それを囲む最も内側の関数リテラル（無ければ fx.Populate の直後）から参照までの間に
// ガードがあるかを見ます。start / stop のように独立して呼ばれうるクロージャが複数ある場合、
// 一方のガードは他方をカバーしないため、クロージャ単位で判定します。
// 同じクロージャ内の複数参照は同一の欠落なので、最初の 1 件だけを報告します。
func collectUnguardedFxPopulate(lines []string, file string) (int, int, []string) {
	code := stripCommentsAndRawStrings(lines)

	var violations []string
	sites, analyzed := 0, 0

	for i, line := range code {
		if !strings.Contains(line, fxPopulateCall) {
			continue
		}
		sites++

		end := funcEndLine(code, i)

		call, callEnd := joinCall(code, i, fxPopulateCall)
		names := populatedNames(call)
		appVar, newAt := fxNewSite(code, funcStartLine(code, i), end)
		if len(names) == 0 || appVar == "" {
			continue
		}
		analyzed++

		// fx.Populate は fx.New の引数なので、対象名は構築式そのものにも現れる。
		// 走査は構築式が閉じた後から始め、引数の記述を参照と取り違えないようにする。
		_, newEnd := joinCall(code, newAt, fxNewCall)
		scanFrom := max(callEnd, newEnd)

		violations = append(violations, unguardedUses(code, lines, file, scanFrom, end, appVar, names)...)
	}

	return sites, analyzed, violations
}

// unguardedUses は、fx.Populate（populateAt 行）から関数終端 end までの間で、
// ガード（`<appVar>.Err()`）に先行されていない populate 対象の参照を列挙します。
func unguardedUses(code, lines []string, file string, populateAt, end int, appVar string, names []string) []string {
	var violations []string
	guard := appVar + ".Err()"
	reported := make(map[int]bool)

	for use := populateAt + 1; use < end && use < len(code); use++ {
		if !referencesAny(code[use], names) {
			continue
		}
		scope := enclosingClosure(code, populateAt, use)
		if reported[scope] {
			continue
		}
		if hasGuardBefore(code, populateAt, use, scope, guard) {
			continue
		}
		reported[scope] = true
		violations = append(violations,
			fmt.Sprintf("%s:%d: %s", file, use+1, strings.TrimSpace(lines[use])))
	}

	return violations
}

// hasGuardBefore は、use 行より前にあり、かつ use を必ず先行実行するガードが存在するかを返します。
//
// ガードが関数直下（囲む関数リテラルなし）にあれば、後続のどのクロージャの参照も先行実行されます。
// クロージャの中にあるガードは、同じクロージャ内の参照しかカバーしません（別のクロージャは独立に
// 呼ばれうるため）。入れ子は最も内側のクロージャだけで判定するので、多重入れ子では
// カバーしていても違反とみなす側（保守的）に倒れます。
func hasGuardBefore(code []string, populateAt, use, useScope int, guard string) bool {
	for g := populateAt + 1; g < use; g++ {
		if !strings.Contains(code[g], guard) {
			continue
		}
		guardScope := enclosingClosure(code, populateAt, g)
		if guardScope == -1 || guardScope == useScope {
			return true
		}
	}
	return false
}

// enclosingClosure は、line を囲む最も内側の関数リテラルの開始行を返します。
// bound より外側は見ず、関数直下（囲む関数リテラルが無い）の場合は -1 を返します。
//
// ブロックが外側へ出るほどインデントは浅くなるので、line から遡って「それまでの最小インデント
// より浅い行」だけを外側のブロック開始とみなします。空行はインデントを持たないため飛ばします。
func enclosingClosure(code []string, bound, line int) int {
	if line >= len(code) {
		return -1
	}
	minIndent := indentWidth(code[line])

	for j := line - 1; j > bound; j-- {
		if strings.TrimSpace(code[j]) == "" {
			continue
		}
		indent := indentWidth(code[j])
		if indent >= minIndent {
			continue
		}
		minIndent = indent
		if closureStartRe.MatchString(strings.TrimRight(code[j], " \t")) {
			return j
		}
	}
	return -1
}

// indentWidth は、行頭のタブ数を返します（gofmt はインデントにタブのみを使います）。
func indentWidth(line string) int {
	return len(line) - len(strings.TrimLeft(line, "\t"))
}

// fxNewSite は、[from, to] の範囲で fx.New の戻り値を受ける変数名とその行を返します。
// 変数で受けていない（戻り値を直接返している等）場合は空文字と -1 を返します。
// fx.Populate は fx.New の引数なので、呼び出し行は fx.New より前にも後にも来ます。
// そのため範囲は fx.Populate の位置ではなく関数全体を渡します。
func fxNewSite(code []string, from, to int) (string, int) {
	for i := from; i <= to && i < len(code); i++ {
		if !strings.Contains(code[i], fxNewCall) {
			continue
		}
		if m := fxNewAssignRe.FindStringSubmatch(code[i]); m != nil {
			return m[1], i
		}
	}
	return "", -1
}

// funcStartLine は、line を含む関数宣言の開始行（行頭 `func `）を返します。
// 見つからない場合は 0 を返します。
func funcStartLine(code []string, line int) int {
	for i := line; i >= 0 && i < len(code); i-- {
		if strings.HasPrefix(code[i], "func ") {
			return i
		}
	}
	return 0
}

// joinCall は、from 行に現れる call の開き括弧から対応する閉じ括弧までを 1 つの文字列に連結し、
// 閉じ括弧のある行を返します。gofmt は引数を複数行に分けた形をそのまま保つため、
// 単一行前提だと取りこぼします。呼び出しが見つからない / 括弧が閉じない場合は空文字と from を返します。
func joinCall(code []string, from int, call string) (string, int) {
	if from < 0 || from >= len(code) {
		return "", from
	}
	idx := strings.Index(code[from], call)
	if idx < 0 {
		return "", from
	}

	var b strings.Builder
	depth := 0
	for i := from; i < len(code); i++ {
		line := code[i]
		start := 0
		if i == from {
			start = idx + len(call) - 1 // 開き括弧から読み始める
		}
		for _, r := range line[start:] {
			switch r {
			case '(':
				depth++
			case ')':
				depth--
			}
			b.WriteRune(r)
			if depth == 0 {
				return call[:len(call)-1] + b.String(), i
			}
		}
	}
	return "", from
}

// populatedNames は、`fx.Populate(&a, &b)` の呼び出し文字列から populate 対象の識別子を取り出します。
// すべての引数が `&識別子` の直接形である場合だけ名前を返し、`fx.Annotate(...)` などで
// 包まれていて対象を一意に特定できない場合は nil を返します（解析対象から外す）。
func populatedNames(call string) []string {
	open := strings.Index(call, "(")
	if open < 0 || !strings.HasSuffix(call, ")") {
		return nil
	}
	args := strings.TrimSpace(call[open+1 : len(call)-1])
	if args == "" {
		return nil
	}

	parts := strings.Split(args, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		// 複数行で書いた呼び出しは末尾にカンマが残るため、空の要素は区切りの余りとして読み飛ばす。
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		name, ok := strings.CutPrefix(trimmed, "&")
		if !ok || !identifierRe.MatchString(name) {
			return nil
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

// stripCommentsAndRawStrings は、行数を保ったままコメントと raw string の中身を空白へ置き換えた
// ソースを返します。対象名を字面で含むコメントや、`}` を含む埋め込みテンプレートが走査を
// 誤らせるため、判定はこの view に対して行います。
func stripCommentsAndRawStrings(lines []string) []string {
	out := make([]string, len(lines))
	var s stripState

	for i, line := range lines {
		var b strings.Builder
		s.inQuote = false
		for j := 0; j < len(line); {
			j = s.step(&b, line, j)
		}
		out[i] = b.String()
	}
	return out
}

// step は line の j バイト目を処理し、出力へ書き込んだうえで次に読む位置を返します。
func (s *stripState) step(b *strings.Builder, line string, j int) int {
	switch {
	case s.inRaw:
		return s.stepRaw(b, line, j)
	case s.inBlock:
		return s.stepBlock(b, line, j)
	case s.inQuote:
		return s.stepQuote(b, line, j)
	default:
		return s.stepCode(b, line, j)
	}
}

// stepRaw は、raw string の内部を空白へ置き換えつつ、閉じるバッククォートを検出します。
func (s *stripState) stepRaw(b *strings.Builder, line string, j int) int {
	if line[j] == '`' {
		s.inRaw = false
		b.WriteByte('`')
		return j + 1
	}
	b.WriteByte(' ')
	return j + 1
}

// stepBlock は、ブロックコメントの内部を空白へ置き換えつつ、閉じる `*/` を検出します。
func (s *stripState) stepBlock(b *strings.Builder, line string, j int) int {
	if line[j] == '*' && byteAt(line, j+1) == '/' {
		s.inBlock = false
		b.WriteString("  ")
		return j + 2
	}
	b.WriteByte(' ')
	return j + 1
}

// stepQuote は、文字列リテラルの内部をそのまま写し、エスケープを飛ばして閉じ `"` を検出します。
func (s *stripState) stepQuote(b *strings.Builder, line string, j int) int {
	c := line[j]
	b.WriteByte(c)
	if c == '\\' && j+1 < len(line) {
		b.WriteByte(line[j+1])
		return j + 2
	}
	if c == '"' {
		s.inQuote = false
	}
	return j + 1
}

// stepCode は、コード領域を写しつつリテラル・コメントの開始を検出します。
// 行コメントは以降を丸ごと落とすため、行末位置を返します。
func (s *stripState) stepCode(b *strings.Builder, line string, j int) int {
	c, next := line[j], byteAt(line, j+1)
	switch {
	case c == '/' && next == '/':
		return len(line)
	case c == '/' && next == '*':
		s.inBlock = true
		b.WriteString("  ")
		return j + 2
	case c == '"':
		s.inQuote = true
	case c == '`':
		s.inRaw = true
	}
	b.WriteByte(c)
	return j + 1
}

// funcEndLine は、from 行を含む関数の終端（行頭 `}`）の行インデックスを返します。
// gofmt は関数の閉じ波括弧を行頭に置くため、次の行頭 `}` までを関数本体とみなせます。
// 見つからない場合はファイル末尾を返します。
func funcEndLine(code []string, from int) int {
	for i := from; i < len(code); i++ {
		if strings.HasPrefix(code[i], "}") {
			return i
		}
	}
	return len(code) - 1
}

// referencesAny は、行が names のいずれかを識別子として含むかを返します。
// 部分一致で別の識別子（`logger` に対する `loggerFactory` など）を拾わないよう前後の文字を見ます。
func referencesAny(line string, names []string) bool {
	for _, name := range names {
		if containsIdentifier(line, name) {
			return true
		}
	}
	return false
}

// containsIdentifier は、line 中に name が識別子として現れるかを返します。
// 前後が Go の識別子構成文字（英数字 / `_`）でない出現だけを識別子とみなします。
func containsIdentifier(line, name string) bool {
	if name == "" {
		return false
	}
	for offset := 0; ; {
		idx := strings.Index(line[offset:], name)
		if idx < 0 {
			return false
		}
		at := offset + idx
		if !isIdentifierChar(byteAt(line, at-1)) && !isIdentifierChar(byteAt(line, at+len(name))) {
			return true
		}
		offset = at + len(name)
	}
}

// byteAt は、範囲外を 0 として line の i バイト目を返します。
func byteAt(line string, i int) byte {
	if i < 0 || i >= len(line) {
		return 0
	}
	return line[i]
}

// isIdentifierChar は、Go の識別子を構成しうるバイトかを返します。
func isIdentifierChar(b byte) bool {
	return b == '_' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

func Test_collectUnguardedFxPopulate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("参照より前にErr検査があれば違反にしない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar a A\n\tapp := fx.New(fx.Populate(&a))\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn a.Do()\n}\n"
			sites, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, sites)
			assert.Equal(t, 1, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("populate対象を一度も参照しなければ検査の有無を問わない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() *fx.App {\n\tvar a A\n\tapp := fx.New(fx.Populate(&a))\n\treturn app\n}\n"
			_, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("引数を複数行に分けた呼び出しも解析できる", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar (\n\t\ta A\n\t\tb B\n\t)\n" +
				"\tapp := fx.New(\n\t\tfx.Populate(\n\t\t\t&a,\n\t\t\t&b,\n\t\t),\n\t)\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn a.Do(b)\n}\n"
			_, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("クロージャごとに検査していれば違反にしない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() (func() error, func() error) {\n\tvar logger L\n" +
				"\tapp := fx.New(fx.Populate(&logger))\n\n" +
				"\tstart := func() error {\n\t\tif err := app.Err(); err != nil {\n\t\t\treturn err\n\t\t}\n" +
				"\t\treturn logger.Do()\n\t}\n\n" +
				"\tstop := func() error {\n\t\tif err := app.Err(); err != nil {\n\t\t\treturn err\n\t\t}\n" +
				"\t\treturn logger.Close()\n\t}\n\n\treturn start, stop\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Empty(t, violations)
		})

		t.Run("関数直下の検査は後続のクロージャもカバーする", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() func() error {\n\tvar logger L\n" +
				"\tapp := fx.New(fx.Populate(&logger))\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn nil\n\t}\n\n" +
				"\treturn func() error {\n\t\treturn logger.Do()\n\t}\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Empty(t, violations)
		})

		t.Run("対象を特定できない包み方は解析対象から外す", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() {\n\tvar got []T\n" +
				"\tapp := fx.New(fx.Populate(fx.Annotate(&got, fx.ParamTags(\"g\"))))\n\tuse(app, got)\n}\n"
			sites, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, sites)
			assert.Zero(t, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("fx.Newを変数で受けていない呼び出しは解析対象から外す", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() *fx.App {\n\tvar a A\n\treturn fx.New(fx.Populate(&a))\n}\n"
			sites, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, sites)
			assert.Zero(t, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("対象名を字面で含むコメントは参照とみなさない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar logger L\n\tapp := fx.New(fx.Populate(&logger))\n" +
				"\tsetup() // logger を触る前に構築エラーを返す\n\t/* logger */\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n" +
				"\treturn logger.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Empty(t, violations)
		})

		t.Run("対象名を部分文字列として含む別の識別子は参照とみなさない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar logger L\n\tapp := fx.New(fx.Populate(&logger))\n" +
				"\tloggerFactory.Warm()\n\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n" +
				"\treturn logger.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Empty(t, violations)
		})

		t.Run("fx.Populateを含まないソースは呼び出し0件で違反なし", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\treturn nil\n}\n"
			sites, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Zero(t, sites)
			assert.Zero(t, analyzed)
			assert.Empty(t, violations)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Err検査が無いまま参照していると検出する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar a A\n\tapp := fx.New(fx.Populate(&a))\n" +
				"\tif err := app.Start(); err != nil {\n\t\treturn err\n\t}\n\treturn a.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:9: return a.Do()", violations[0])
		})

		t.Run("Err検査が参照より後ろにあると検出する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar a A\n\tapp := fx.New(fx.Populate(&a))\n" +
				"\ta.Warm()\n\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:6: a.Warm()", violations[0])
		})

		t.Run("別レシーバのErr呼び出しはガードとみなさない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar logger L\n\tapp := fx.New(fx.Populate(&logger))\n" +
				"\tif err := resp.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn logger.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:9: return logger.Do()", violations[0])
		})

		t.Run("片方のクロージャだけ検査していると他方を検出する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() (func() error, func() error) {\n\tvar logger L\n" +
				"\tapp := fx.New(fx.Populate(&logger))\n\n" +
				"\tstart := func() error {\n\t\tif err := app.Err(); err != nil {\n\t\t\treturn err\n\t\t}\n" +
				"\t\treturn logger.Do()\n\t}\n\n" +
				"\tstop := func() error {\n\t\treturn logger.Close()\n\t}\n\n\treturn start, stop\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:15: return logger.Close()", violations[0])
		})

		t.Run("raw string内の行頭閉じ波括弧で走査を打ち切らない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar logger L\n\tapp := fx.New(fx.Populate(&logger))\n" +
				"\tconst tmpl = `\n}\n`\n\t_ = app\n\treturn logger.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:10: return logger.Do()", violations[0])
		})

		t.Run("複数対象のうち一つでも検査前に参照していると検出する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar (\n\t\ta A\n\t\tb B\n\t)\n" +
				"\tapp := fx.New(fx.Populate(&a, &b))\n\tb.Warm()\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn a.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:9: b.Warm()", violations[0])
		})

		t.Run("同じクロージャ内の複数参照は最初の1件だけ報告する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar a A\n\tapp := fx.New(fx.Populate(&a))\n" +
				"\ta.Warm()\n\ta.Do()\n\t_ = app\n\treturn nil\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:6: a.Warm()", violations[0])
		})
	})
}

func Test_unguardedUses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ガード済みなら空を返す", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tfx.Populate(&a)\n\tif err := app.Err(); err != nil {\n\t}\n\ta.Do()\n}", "\n")
			assert.Empty(t, unguardedUses(lines, lines, "p.go", 1, 5, "app", []string{"a"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ガードが無ければ参照行を返す", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tfx.Populate(&a)\n\ta.Do()\n}", "\n")
			assert.Equal(t, []string{"p.go:3: a.Do()"}, unguardedUses(lines, lines, "p.go", 1, 3, "app", []string{"a"}))
		})
	})
}

func Test_hasGuardBefore(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一スコープのガードを認める", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tapp := fx.New()\n\tif err := app.Err(); err != nil {\n\t}\n\ta.Do()", "\n")
			assert.True(t, hasGuardBefore(lines, 1, 4, -1, "app.Err()"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("別クロージャ内のガードは認めない", func(t *testing.T) {
			t.Parallel()
			src := "x\n\tapp := fx.New()\n\tstart := func() error {\n\t\tif err := app.Err(); err != nil {\n\t\t}\n\t}\n" +
				"\tstop := func() error {\n\t\ta.Do()\n\t}"
			lines := strings.Split(src, "\n")
			assert.False(t, hasGuardBefore(lines, 1, 7, 6, "app.Err()"))
		})

		t.Run("ガードが無ければ偽", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tapp := fx.New()\n\ta.Do()", "\n")
			assert.False(t, hasGuardBefore(lines, 1, 2, -1, "app.Err()"))
		})
	})
}

func Test_enclosingClosure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("囲む関数リテラルの開始行を返す", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tapp := fx.New()\n\tstart := func() error {\n\n\t\treturn a.Do()\n\t}", "\n")
			assert.Equal(t, 2, enclosingClosure(lines, 1, 4))
		})

		t.Run("関数直下なら-1を返す", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tapp := fx.New()\n\treturn a.Do()", "\n")
			assert.Equal(t, -1, enclosingClosure(lines, 1, 2))
		})

		t.Run("if ブロックは関数リテラルとみなさない", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("x\n\tapp := fx.New()\n\tif ok {\n\t\treturn a.Do()\n\t}", "\n")
			assert.Equal(t, -1, enclosingClosure(lines, 1, 3))
		})

		t.Run("範囲外の行は-1を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, -1, enclosingClosure([]string{"a"}, 0, 5))
		})
	})
}

func Test_indentWidth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行頭のタブ数を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 0, indentWidth("a"))
			assert.Equal(t, 2, indentWidth("\t\ta"))
		})
	})
}

func Test_fxNewSite(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("代入先の変数名とその行を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"\tapp := fx.New(fx.Populate(&a))"}
			name, at := fxNewSite(lines, 0, 0)
			assert.Equal(t, "app", name)
			assert.Equal(t, 0, at)
		})

		t.Run("fx.Populateより前にあるfx.Newも拾う", func(t *testing.T) {
			t.Parallel()
			lines := []string{"\tapp := fx.New(", "\t\tfx.Populate(&a),", "\t)"}
			name, at := fxNewSite(lines, 0, 2)
			assert.Equal(t, "app", name)
			assert.Equal(t, 0, at)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("変数で受けていなければ空文字と-1を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"\treturn fx.New(fx.Populate(&a))"}
			name, at := fxNewSite(lines, 0, 0)
			assert.Empty(t, name)
			assert.Equal(t, -1, at)
		})
	})
}

func Test_funcStartLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行頭のfunc宣言を関数の開始とする", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() {\n\tg()\n}\n", "\n")
			assert.Equal(t, 2, funcStartLine(lines, 3))
		})

		t.Run("宣言が無ければ0を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 0, funcStartLine([]string{"package p", "\tg()"}, 1))
		})
	})
}

func Test_joinCall(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一行の呼び出しをそのまま返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"\tapp := fx.New(fx.Populate(&a, &b))"}
			call, at := joinCall(lines, 0, fxPopulateCall)
			assert.Equal(t, "fx.Populate(&a, &b)", call)
			assert.Equal(t, 0, at)
		})

		t.Run("複数行の呼び出しを連結し閉じ括弧の行を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"\t\tfx.Populate(", "\t\t\t&a,", "\t\t\t&b,", "\t\t),"}
			call, at := joinCall(lines, 0, fxPopulateCall)
			assert.Equal(t, "fx.Populate(\t\t\t&a,\t\t\t&b,\t\t)", call)
			assert.Equal(t, 3, at)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("呼び出しを含まない行は空文字を返す", func(t *testing.T) {
			t.Parallel()
			call, _ := joinCall([]string{"\tapp := fx.New(opts...)"}, 0, fxPopulateCall)
			assert.Empty(t, call)
		})

		t.Run("括弧が閉じなければ空文字を返す", func(t *testing.T) {
			t.Parallel()
			call, _ := joinCall([]string{"\tfx.Populate("}, 0, fxPopulateCall)
			assert.Empty(t, call)
		})

		t.Run("範囲外の行は空文字を返す", func(t *testing.T) {
			t.Parallel()
			call, _ := joinCall([]string{"\tfx.Populate(&a)"}, 5, fxPopulateCall)
			assert.Empty(t, call)
		})
	})
}

func Test_populatedNames(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アドレス演算子付きの引数から識別子を取り出す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "b"}, populatedNames("fx.Populate(&a, &b)"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フィールド参照は対象を特定できないためnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("fx.Populate(&deps.Logger)"))
		})

		t.Run("アドレス演算子が無い引数が混じればnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("fx.Populate(&a, b)"))
		})

		t.Run("引数が空ならnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("fx.Populate()"))
		})

		t.Run("呼び出しの形になっていなければnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames(""))
		})
	})
}

func Test_stripCommentsAndRawStrings(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行コメントを落とす", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"\tsetup() "}, stripCommentsAndRawStrings([]string{"\tsetup() // logger"}))
		})

		t.Run("ブロックコメントを空白に置き換える", func(t *testing.T) {
			t.Parallel()
			got := stripCommentsAndRawStrings([]string{"\ta := 1 /* logger */ + 2"})
			assert.False(t, containsIdentifier(got[0], "logger"))
			assert.True(t, containsIdentifier(got[0], "a"))
		})

		t.Run("raw stringの中身を空白に置き換える", func(t *testing.T) {
			t.Parallel()
			got := stripCommentsAndRawStrings([]string{"\tconst t = `", "}", "`"})
			assert.False(t, strings.HasPrefix(got[1], "}"))
		})

		t.Run("文字列リテラル内のスラッシュはコメントとみなさない", func(t *testing.T) {
			t.Parallel()
			got := stripCommentsAndRawStrings([]string{"\turl := \"http://x\" + logger.Host()"})
			assert.True(t, containsIdentifier(got[0], "logger"))
		})

		t.Run("行数を保つ", func(t *testing.T) {
			t.Parallel()
			in := []string{"a", "b", "c"}
			assert.Len(t, stripCommentsAndRawStrings(in), len(in))
		})
	})
}

func Test_funcEndLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行頭の閉じ波括弧を関数の終端とする", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() {\n\tg()\n}\n\nfunc h() {}\n", "\n")
			assert.Equal(t, 4, funcEndLine(lines, 3))
		})

		t.Run("終端が無ければファイル末尾を返す", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() {\n\tg()\n", "\n")
			assert.Equal(t, len(lines)-1, funcEndLine(lines, 3))
		})
	})
}

func Test_referencesAny(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("識別子として現れていれば真", func(t *testing.T) {
			t.Parallel()
			assert.True(t, referencesAny("\treturn logger.Do()", []string{"logger"}))
		})

		t.Run("部分文字列でしかなければ偽", func(t *testing.T) {
			t.Parallel()
			assert.False(t, referencesAny("\tloggerFactory.Warm()", []string{"logger"}))
		})

		t.Run("候補が空なら偽", func(t *testing.T) {
			t.Parallel()
			assert.False(t, referencesAny("\treturn nil", nil))
		})
	})
}

func Test_containsIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("識別子として現れていれば真", func(t *testing.T) {
			t.Parallel()
			assert.True(t, containsIdentifier("a := a + 1", "a"))
		})

		t.Run("前後が識別子構成文字なら偽", func(t *testing.T) {
			t.Parallel()
			assert.False(t, containsIdentifier("abc := 1", "a"))
		})

		t.Run("最初の出現が部分一致でも後続に識別子があれば真", func(t *testing.T) {
			t.Parallel()
			assert.True(t, containsIdentifier("abc := a", "a"))
		})

		t.Run("空の名前は偽", func(t *testing.T) {
			t.Parallel()
			assert.False(t, containsIdentifier("abc", ""))
		})
	})
}

func Test_byteAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("範囲内はそのバイトを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, byte('b'), byteAt("abc", 1))
		})

		t.Run("範囲外は0を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, byte(0), byteAt("abc", -1))
			assert.Equal(t, byte(0), byteAt("abc", 3))
		})
	})
}

func Test_isIdentifierChar(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("英数字とアンダースコアは真", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isIdentifierChar('a'))
			assert.True(t, isIdentifierChar('Z'))
			assert.True(t, isIdentifierChar('0'))
			assert.True(t, isIdentifierChar('_'))
		})

		t.Run("区切り文字は偽", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isIdentifierChar('.'))
			assert.False(t, isIdentifierChar(' '))
			assert.False(t, isIdentifierChar(0))
		})
	})
}
