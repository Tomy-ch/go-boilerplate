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

// TestIdempotencyCompleteness は、OpenAPI で Idempotency-Key ヘッダを宣言した操作
// （= 生成コードの <Op>Params 型が IdempotencyKey フィールドを持つ操作）のハンドラが、
// 必ず idempotency.Run 経由で処理していることを機械検証する完全性テストです。
//
// 冪等性は「変更系すべて」ではなく「Idempotency-Key を宣言した操作」に閉じる（PUT/DELETE は
// HTTP 的に冪等で Run を要さない）ため、OpenAPI のヘッダ宣言を唯一のマーカー（source of truth）と
// して扱う。middleware 登録済みで Run 呼び忘れ、のような silent な dedup 欠落を loud な失敗に変える。
//
// Idempotency-Key を宣言する操作は 0 件でも許容する（サンプル API 削除後は該当操作が無くなり得る）。
// ただし検出ロジック自体が陳腐化して空振りしていないことは保証したいので、`<Op>Params struct` を
// 生成コードから最低 1 件は検出できること（= 正規表現が生きていること）を別途 assert する。これは
// IdempotencyKey フィールドの有無とは独立した健全性チェックで、コアの GetExchangeRatesParams が
// 常に存在するため成立する。
//
// 本リポジトリは depguard で go/ast 等のツールチェーンパッケージを禁止するため、AST ではなく
// gofmt 済みソースのテキスト走査で検出する（`type ...Params struct {` / `func (recv) Name(` は
// gofmt により行頭固定なので、この走査は安定する）。
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
	// 常に成功してしまう。IdempotencyKey の有無に依らず `<Op>Params struct` を最低 1 件は
	// 検出できること（正規表現が生きていること）を保証する。marked が空（冪等操作 0 件）でも可。
	require.Positive(t, totalParamsSeen, "生成コードから `<Op>Params struct` を1件も検出できない（正規表現の陳腐化を疑う）")

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
