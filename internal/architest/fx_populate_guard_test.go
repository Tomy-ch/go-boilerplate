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

const fxPopulateCall = "fx.Populate("

var (
	// fxPopulateArgsRe は、`fx.Populate(&a, &b)` の引数列を取り出します。gofmt は 1 呼び出しを
	// 1 行に収めるため、閉じ括弧までを同一行から取れる前提で十分です。
	fxPopulateArgsRe = regexp.MustCompile(`fx\.Populate\(([^)]*)\)`)
	// fxErrGuardRe は、構築エラーの検査（`app.Err()` など任意のレシーバの `.Err()` 呼び出し）にマッチします。
	fxErrGuardRe = regexp.MustCompile(`\.Err\(\)`)
)

// TestFxPopulateHasConstructionErrorGuard は、fx.Populate を使う production 関数が、populate 対象を
// 参照する前に構築エラーを検査していることを機械検証する。
//
// fx.Populate は fx.New 時点の invoke であり、グラフ構築が失敗すると対象は nil のまま残る。
// そのまま参照すれば nil 参照で落ちる。fx.App.Start がエラーを短絡して返すため「Start を先に
// 呼んでいる限り安全」という状態は成立しうるが、それは実装順序という暗黙の前提にすぎず、
// コンパイラも lint も守らない。参照より前に `.Err()` を検査する形を構造として固定し、
// 呼び出し箇所ごとに防御の有無が割れないようにする。
//
// 検出は depguard が go/ast を禁じるためテキスト走査で行う（既存 architest と同方針）。
// 解析対象は `fx.Populate(&a, &b)` の直接形に限り、`fx.Annotate` 等で包んだ形は対象を特定できない
// ため検査しない。見送った件数はログに出し、カバー範囲が黙って狭まらないようにする。
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
	t.Logf("fx.Populate 呼び出し数: %d / 解析対象: %d / violations: %d", sites, analyzed, len(violations))

	require.Empty(t, violations,
		"fx.Populate の対象を構築エラーの検査より前に参照している。populate 対象に触れる前に "+
			"`if err := app.Err(); err != nil { ... }` で構築エラーを返すこと。")
}

// collectUnguardedFxPopulate は、gofmt 済みソースの行列から fx.Populate の呼び出し数・そのうち
// 解析できた数・populate 対象を `.Err()` 検査より前に参照している箇所（`file:line: 該当行` 形式）を返します。
//
// 判定は fx.Populate の呼び出し行を起点に、それを含む関数の終わりまでを走査し、最初の `.Err()`
// 検査と、最初の populate 対象参照の前後関係を見ます。検査が無い場合も違反です。
// 対象名を字面として含むだけの行コメントは参照とみなしません。
func collectUnguardedFxPopulate(lines []string, file string) (int, int, []string) {
	var violations []string
	sites, analyzed := 0, 0

	for i, line := range lines {
		if !strings.Contains(line, fxPopulateCall) || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		sites++

		names := populatedNames(line)
		if len(names) == 0 {
			continue
		}
		analyzed++

		end := funcEndLine(lines, i)
		guard := firstMatchLine(lines, i+1, end, func(l string) bool { return fxErrGuardRe.MatchString(l) })
		use := firstMatchLine(lines, i+1, end, func(l string) bool { return referencesAny(l, names) })

		// 参照が一切無ければ nil を触りようがないので、検査の有無を問わない。
		if use < 0 {
			continue
		}
		if guard < 0 || guard > use {
			violations = append(violations,
				fmt.Sprintf("%s:%d: %s", file, use+1, strings.TrimSpace(lines[use])))
		}
	}

	return sites, analyzed, violations
}

// populatedNames は、`fx.Populate(&a, &b)` の引数から populate 対象の識別子を取り出します。
// すべての引数が `&識別子` の直接形である場合だけ名前を返し、`fx.Annotate(...)` などで
// 包まれていて対象を一意に特定できない場合は nil を返します（解析対象から外す）。
func populatedNames(line string) []string {
	m := fxPopulateArgsRe.FindStringSubmatch(line)
	if m == nil || strings.TrimSpace(m[1]) == "" {
		return nil
	}

	args := strings.Split(m[1], ",")
	names := make([]string, 0, len(args))
	for _, arg := range args {
		name, ok := strings.CutPrefix(strings.TrimSpace(arg), "&")
		if !ok || !isIdentifier(name) {
			return nil
		}
		names = append(names, name)
	}
	return names
}

// isIdentifier は、s が Go の識別子（数字始まりでない英数字と `_` の並び）かを返します。
func isIdentifier(s string) bool {
	if s == "" || (s[0] >= '0' && s[0] <= '9') {
		return false
	}
	for i := range len(s) {
		if !isIdentifierChar(s[i]) {
			return false
		}
	}
	return true
}

// funcEndLine は、from 行を含む関数の終端（行頭 `}`）の行インデックスを返します。
// gofmt は関数の閉じ括弧を行頭に置くため、次の行頭 `}` までを関数本体とみなせます。
// 見つからない場合はファイル末尾を返します。
func funcEndLine(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "}") {
			return i
		}
	}
	return len(lines) - 1
}

// firstMatchLine は、[from, to) の範囲で match を満たす最初の行インデックスを返します。
// 行コメントは対象外です。見つからない場合は -1 を返します。
func firstMatchLine(lines []string, from, to int, match func(string) bool) int {
	for i := from; i < to && i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "//") {
			continue
		}
		if match(lines[i]) {
			return i
		}
	}
	return -1
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
			src := "package p\n\nfunc f() *fx.App {\n\tvar a A\n\treturn fx.New(fx.Populate(&a))\n}\n"
			_, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("対象を特定できない包み方は解析対象から外す", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() {\n\tvar got []T\n" +
				"\tapp := fx.New(fx.Populate(fx.Annotate(&got, fx.ParamTags(\"g\"))))\n\tuse(got)\n}\n"
			sites, analyzed, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			assert.Equal(t, 1, sites)
			assert.Zero(t, analyzed)
			assert.Empty(t, violations)
		})

		t.Run("対象名を字面で含む行コメントは参照とみなさない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar logger L\n\tapp := fx.New(fx.Populate(&logger))\n" +
				"\t// logger を触る前に構築エラーを返す。\n\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n" +
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

		t.Run("クロージャ内で参照より前に検査していれば違反にしない", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() func() error {\n\tvar logger L\n" +
				"\tapp := fx.New(\n\t\tfx.Populate(&logger),\n\t\tfx.WithLogger(nil),\n\t)\n\n" +
				"\treturn func() error {\n\t\tif err := app.Err(); err != nil {\n\t\t\treturn err\n\t\t}\n\n" +
				"\t\treturn logger.Do()\n\t}\n}\n"
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

		t.Run("複数対象のうち一つでも検査前に参照していると検出する", func(t *testing.T) {
			t.Parallel()
			src := "package p\n\nfunc f() error {\n\tvar (\n\t\ta A\n\t\tb B\n\t)\n" +
				"\tapp := fx.New(fx.Populate(&a, &b))\n\tb.Warm()\n" +
				"\tif err := app.Err(); err != nil {\n\t\treturn err\n\t}\n\treturn a.Do()\n}\n"
			_, _, violations := collectUnguardedFxPopulate(strings.Split(src, "\n"), "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, "p.go:9: b.Warm()", violations[0])
		})
	})
}

func Test_populatedNames(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アドレス演算子付きの引数から識別子を取り出す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "b"}, populatedNames("\tapp := fx.New(fx.Populate(&a, &b))"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フィールド参照は対象を特定できないためnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("\tfx.Populate(&deps.Logger),"))
		})

		t.Run("アドレス演算子が無い引数が混じればnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("\tfx.Populate(&a, b),"))
		})

		t.Run("引数が空ならnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("\tfx.Populate(),"))
		})

		t.Run("fx.Populateを含まない行はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, populatedNames("\tapp := fx.New(opts...)"))
		})
	})
}

func Test_isIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("英数字とアンダースコアの並びは識別子", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isIdentifier("appCfg"))
			assert.True(t, isIdentifier("_x1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字・数字始まり・区切りを含む文字列は識別子でない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isIdentifier(""))
			assert.False(t, isIdentifier("1a"))
			assert.False(t, isIdentifier("a.b"))
		})
	})
}

func Test_funcEndLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行頭の閉じ括弧を関数の終端とする", func(t *testing.T) {
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

func Test_firstMatchLine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("範囲内で最初に一致した行を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"a", "b", "c", "b"}
			assert.Equal(t, 1, firstMatchLine(lines, 0, len(lines), func(l string) bool { return l == "b" }))
		})

		t.Run("行コメントは一致対象から外す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"// b", "b"}
			assert.Equal(t, 1, firstMatchLine(lines, 0, len(lines), func(l string) bool { return strings.Contains(l, "b") }))
		})

		t.Run("一致が無ければ-1を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"a", "c"}
			assert.Equal(t, -1, firstMatchLine(lines, 0, len(lines), func(l string) bool { return l == "b" }))
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
