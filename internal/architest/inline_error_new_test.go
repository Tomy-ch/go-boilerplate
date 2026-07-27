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

// xerrorsNewCall は、検出対象となる呼び出しの字面です。
const xerrorsNewCall = "xerrors.New("

// sentinelDeclRe は、package-level の単独 var 宣言（`var errXxx = xerrors.New(`）にマッチします。
var sentinelDeclRe = regexp.MustCompile(`^var \w+ = xerrors\.New\(`)

// TestNoInlineXerrorsNew は、production コードが関数本体内で xerrors.New を直接生成していないことを
// 機械検証する。関数本体で生成したエラーは呼び出し側から errors.Is で識別できず、テストが
// メッセージ文字列一致でしか検証できない脆いアサーションに退行するため、package-level の
// sentinel（`var errXxx = xerrors.New(...)`）として宣言し、動的な文脈は xerrors.Wrap で付与する。
//
// 検出は depguard が go/ast を禁じるためテキスト走査で行う（既存 architest と同方針）。
// allowlist は持たない。_test.go はテストが注入用のアドホックなエラーを作る正当な用法のため対象外。
func TestNoInlineXerrorsNew(t *testing.T) {
	t.Parallel()

	var violations []string
	scanned := 0

	for _, root := range moduleSubdirs(t, "internal", "pkg", "cmd", "scripts") {
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
			scanned++
			violations = append(violations, collectInlineXerrorsNew(lines, path)...)
			return nil
		})
		require.NoError(t, err)
	}

	// 走査対象が 0 件だと検証が常に成功してしまうため、ルート解決の破綻を空振りとして検出する。
	require.NotZero(t, scanned, "走査対象の production Go ファイルが 0 件（moduleSubdirs のルート解決を疑う）")

	sort.Strings(violations)
	for _, v := range violations {
		t.Log("inline xerrors.New: " + v)
	}
	t.Logf("走査ファイル数: %d / inline violations: %d", scanned, len(violations))

	require.Empty(t, violations,
		"関数本体内で xerrors.New を生成している。package-level の sentinel を宣言し、"+
			"動的な文脈は xerrors.Wrap(errXxx, ...) で付与すること。")
}

// collectInlineXerrorsNew は、gofmt 済みソースの行列から、package-level の var 宣言以外の位置に
// 現れる xerrors.New 呼び出しを `file:line: 該当行` 形式で列挙します。
//
// gofmt は package-level の var ブロックを行頭 `var (` … 行頭 `)` に整形するため、その区間を
// 許可領域とみなします。関数本体内の var ブロックは行頭がタブになるので許可領域に入りません。
func collectInlineXerrorsNew(lines []string, file string) []string {
	var violations []string
	inVarBlock := false

	for i, line := range lines {
		if inVarBlock {
			if strings.HasPrefix(line, ")") {
				inVarBlock = false
			}
			continue
		}
		if strings.HasPrefix(line, "var (") {
			inVarBlock = true
			continue
		}
		if !strings.Contains(line, xerrorsNewCall) || sentinelDeclRe.MatchString(line) {
			continue
		}
		violations = append(violations, fmt.Sprintf("%s:%d: %s", file, i+1, strings.TrimSpace(line)))
	}
	return violations
}

func Test_collectInlineXerrorsNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("package-levelのvarブロック内の宣言は検出しない", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nvar (\n\terrA = xerrors.New(\"a\")\n\terrB = xerrors.New(\"b\")\n)\n", "\n")
			assert.Empty(t, collectInlineXerrorsNew(lines, "p.go"))
		})

		t.Run("package-levelの単独var宣言は検出しない", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nvar errA = xerrors.New(\"a\")\n", "\n")
			assert.Empty(t, collectInlineXerrorsNew(lines, "p.go"))
		})

		t.Run("引数を改行した単独var宣言は検出しない", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nvar errA = xerrors.New(\n\t\"very long message\")\n", "\n")
			assert.Empty(t, collectInlineXerrorsNew(lines, "p.go"))
		})

		t.Run("xerrors.Newを含まないソースは検出しない", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() error {\n\treturn xerrors.Wrap(errA, \"ctx\")\n}\n", "\n")
			assert.Empty(t, collectInlineXerrorsNew(lines, "p.go"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("関数本体内の生成を検出する", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() error {\n\treturn xerrors.New(\"boom\")\n}\n", "\n")
			violations := collectInlineXerrorsNew(lines, "p.go")
			require.Len(t, violations, 1)
			assert.Equal(t, `p.go:4: return xerrors.New("boom")`, violations[0])
		})

		t.Run("関数本体内のvarブロック内の生成を検出する", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nfunc f() error {\n\tvar (\n\t\terr = xerrors.New(\"boom\")\n\t)\n\treturn err\n}\n", "\n")
			assert.Len(t, collectInlineXerrorsNew(lines, "p.go"), 1)
		})

		t.Run("package-levelのvarブロックを閉じた後の生成を検出する", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split(
				"package p\n\nvar (\n\terrA = xerrors.New(\"a\")\n)\n\nfunc f() error {\n\treturn xerrors.New(\"boom\")\n}\n",
				"\n",
			)
			assert.Len(t, collectInlineXerrorsNew(lines, "p.go"), 1)
		})

		t.Run("package-levelのvar宣言でもsentinel以外への代入は検出する", func(t *testing.T) {
			t.Parallel()
			lines := strings.Split("package p\n\nvar f = func() error { return xerrors.New(\"boom\") }\n", "\n")
			assert.Len(t, collectInlineXerrorsNew(lines, "p.go"), 1)
		})
	})
}
