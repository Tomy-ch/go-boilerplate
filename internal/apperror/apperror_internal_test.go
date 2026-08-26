package apperror

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errDeclRe は、`var ( ... )` ブロック内の `\tErr<Name> = xerrors.New(` 宣言行にマッチし Name を捕捉します。
// gofmt 済みソースでは var ブロックの各要素が 1 タブでインデントされる前提です。
var errDeclRe = regexp.MustCompile(`^\tErr(\w+)\s*=\s*xerrors\.New\(`)

// TestAppErrorsCompleteness は、apperror パッケージで宣言された全 Err* センチネルが、
// worker 分類センチネル（appErrors から意図的に除外）を除いて漏れなく appErrors に列挙されて
// いることを機械検証します。HTTP taxonomy センチネルは var ブロックと appErrors スライスの
// 2 箇所へ二重列挙されており、片方への追加漏れを検出する手段が他にないためです。
func TestAppErrorsCompleteness(t *testing.T) {
	t.Parallel()

	// worker のメッセージ処理分類センチネル。HTTP taxonomy ではないため appErrors には含めない。
	workerSentinels := []error{ErrRetryable, ErrPermanent, ErrFatal}

	declared := declaredErrNames(t)

	// 検出ロジックが命名変更等で空振りすると完全性検証が常に成功してしまう。既知の宣言数（13 app +
	// 3 worker = 16）を下限として直接検証し、正規表現の陳腐化を検出する。
	require.GreaterOrEqual(t, len(declared), 16,
		"Err* 宣言の検出数が想定を下回る（errDeclRe の陳腐化を疑う）: %v", declared)

	require.Len(t, appErrors, len(declared)-len(workerSentinels),
		"appErrors の件数がソース宣言総数と worker センチネル数の差に一致しない（同期漏れを疑う）")

	// appErrors に nil や重複が混入していないことを検証する。
	for i, e := range appErrors {
		require.Errorf(t, e, "appErrors[%d] が nil", i)
	}
	for i := range appErrors {
		for j := i + 1; j < len(appErrors); j++ {
			assert.Falsef(t, xerrors.Is(appErrors[i], appErrors[j]),
				"appErrors に重複センチネルがある: index %d と %d", i, j)
		}
	}

	// worker センチネルは HTTP taxonomy 対象外なので IsAppError は false でなければならない。
	for _, s := range workerSentinels {
		assert.Falsef(t, IsAppError(s), "worker センチネル %v が IsAppError=true（appErrors への誤混入を疑う）", s)
	}
}

// declaredErrNames は、apperror パッケージのソース（_test.go 除く）を走査し、
// `Err<Name> = xerrors.New(` で宣言された全 Err* センチネル名を収集して返します。
// depguard が go/ast を禁止するため、AST ではなく gofmt 済みソースのテキスト走査で検出する。
func declaredErrNames(t *testing.T) []string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(thisFile)

	paths, err := pkgfs.OS{}.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)

	var names []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, readErr := pkgfs.OS{}.ReadFile(path)
		require.NoError(t, readErr)
		for line := range strings.SplitSeq(string(src), "\n") {
			if m := errDeclRe.FindStringSubmatch(line); m != nil {
				names = append(names, "Err"+m[1])
			}
		}
	}
	return names
}
