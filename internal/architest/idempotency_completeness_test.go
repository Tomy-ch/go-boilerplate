// Package architest は、レイヤ横断のアーキテクチャ不変条件を機械検証するテストのみを収容します（実装コードは持ちません）。
package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// paramsStructRe は、生成コードの `type <Op>Params struct {` 行にマッチし Op を捕捉します。
var paramsStructRe = regexp.MustCompile(`^type (\w+)Params struct \{$`)

// handlerMethodRe は、ハンドラのメソッド宣言 `func (recv) <Name>(` にマッチし Name を捕捉します。
var handlerMethodRe = regexp.MustCompile(`^func \([^)]*\) (\w+)\(`)

// TestIdempotencyCompleteness は、OpenAPI で Idempotency-Key を宣言した操作
// （= 生成コードの <Op>Params 型が IdempotencyKey フィールドを持つ操作）のハンドラが、
// 必ず idempotency.Run 経由で処理していることを機械検証する完全性テストです。
// OpenAPI のヘッダ宣言を source of truth とし、Run 呼び忘れのような silent な dedup 欠落を loud な失敗に変える。
//
// 対象操作は 0 件でも許容する（サンプル API 削除後は該当が無くなり得る）が、検出ロジックの空振りを
// 防ぐため、param 検出用の正規表現が既知の生成形にマッチすることを別途 assert する。depguard が
// go/ast を禁止するため、AST ではなく gofmt 済みソースのテキスト走査で検出する。
func TestIdempotencyCompleteness(t *testing.T) {
	t.Parallel()

	root := handlerRoot(t)

	marked := map[string]string{}    // operationID -> 宣言元ファイル（Idempotency-Key を宣言する操作）
	wrapped := map[string]struct{}{} // idempotency.Run を呼ぶハンドラメソッド名
	totalParamsSeen := 0             // 走査中に見た `<Op>Params struct` の総数（検出健全性の指標）

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		src, readErr := pkgfs.OS{}.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lines := strings.Split(string(src), "\n")

		if strings.HasSuffix(path, ".gen.go") {
			totalParamsSeen += collectMarkedParams(lines, path, marked)
		} else {
			collectRunWrapped(lines, wrapped)
		}
		return nil
	})
	require.NoError(t, err)

	// マーカー検出ロジックが生成コードの命名変更等で陳腐化して空振りすると、完全性検証が
	// 常に成功してしまう。これを防ぐため、param 検出の正規表現が既知の生成形にマッチすることを
	// 直接検証する。サンプル API を全削除すると param を持つ操作は 0 件になり得る（totalParamsSeen==0）
	// ため、リポジトリ走査の件数ではなく正規表現の自己検査で陳腐化を検出する。
	t.Logf("param 構造体の検出数: %d（サンプル全削除後は 0 になり得る）", totalParamsSeen)
	require.NotEmpty(t, paramsStructRe.FindStringSubmatch("type XxxParams struct {"),
		"paramsStructRe が `<Op>Params struct {` 形にマッチしない（正規表現の陳腐化を疑う）")

	for op, file := range marked {
		_, ok := wrapped[op]
		assert.Truef(t, ok,
			"操作 %s は Idempotency-Key を宣言している（%s）が、ハンドラが idempotency.Run を呼んでいない", op, file)
	}
}

// handlerRoot は、本テストファイルから見た internal/controller/handler の絶対パスを返します。
func handlerRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/architest/<file> から internal/controller/handler を辿る。
	return filepath.Join(filepath.Dir(thisFile), "..", "controller", "handler")
}

// collectMarkedParams は、`type <Op>Params struct {` ブロックが IdempotencyKey フィールドを
// 含むとき Op を marked へ加えます。gofmt 済みソースでは閉じ括弧が行頭 `}` に現れる前提です。
// 戻り値は、IdempotencyKey の有無に依らず走査中に検出した `<Op>Params struct` の件数で、
// 呼び出し側の検出健全性チェック（正規表現の陳腐化検出）に用います。
func collectMarkedParams(lines []string, file string, marked map[string]string) int {
	seen := 0
	for i, line := range lines {
		m := paramsStructRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		seen++
		op := m[1]
		for _, body := range lines[i+1:] {
			if body == "}" {
				break
			}
			if strings.Contains(body, "IdempotencyKey") {
				marked[op] = file
				break
			}
		}
	}
	return seen
}

// collectRunWrapped は、メソッド宣言ブロック内に idempotency.Run 呼び出しがあるとき、その
// メソッド名を wrapped へ加えます。ブロックは行頭 `func ` から次の行頭 `func ` までとします。
//
// 検出は import 別名なしの直接呼び出し `idempotency.Run(` に限る。別名 import（`idem.Run(` 等）や
// 変数へ束ねてからの間接呼び出しは検出できない。現行ハンドラは直接呼び出しのみのため問題ないが、
// この規約から外れると完全性検証が偽陰性になりうる点に注意。
func collectRunWrapped(lines []string, wrapped map[string]struct{}) {
	currentMethod := ""
	for _, line := range lines {
		if m := handlerMethodRe.FindStringSubmatch(line); m != nil {
			currentMethod = m[1]
			continue
		}
		if strings.HasPrefix(line, "func ") {
			currentMethod = "" // メソッド以外の関数宣言に入ったらリセット
			continue
		}
		if currentMethod != "" && strings.Contains(line, "idempotency.Run(") {
			wrapped[currentMethod] = struct{}{}
		}
	}
}
