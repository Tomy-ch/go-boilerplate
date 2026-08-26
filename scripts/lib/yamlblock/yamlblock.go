// Package yamlblock は YAML のブロックスカラー（`key: |` / `key: >-`）の中身の判定を 1 箇所へ集める。
//
// 呼び出し側は 2 つある。`uses: owner/repo@<sha>` の取りこぼしを検出する pin-actions と、
// `uses: docker://<image>:<tag>` の取りこぼしを検出する pin-images である。どちらも
// 「厳格なパターンで潰した残り」を行単位で走査するため、ブロックスカラーの中身を外さなければ
// `run:` スクリプトが `uses:` を含む文字列を出力するだけで検出が誤爆する。
//
// 片方だけが除外すると、同じ workflow が経路によって違う判定を受ける。
package yamlblock

import (
	"regexp"
	"strings"
)

// headerRe は値がブロックスカラー（`|` / `>`）で始まる行。
// 字下げ指示子と chomp 指示子は YAML がどちらの順序も許すため（`|2-` / `|-2`）両方を受ける。
var headerRe = regexp.MustCompile(`:[ \t]*[|>][+-]?\d?[+-]?[ \t]*(?:#.*)?$`)

// ContentLines は data のうちブロックスカラーの中身に当たる行番号（1 始まり）を返す。
//
// 中身の範囲は字下げで決まる。ヘッダ行より深い字下げの行と、その途中の空行が中身で、
// 字下げがヘッダ以下へ戻った行で終わる。ヘッダ行そのものは中身に含めない。
func ContentLines(data string) map[int]bool {
	content := map[int]bool{}
	headerIndent := -1

	for i, line := range strings.Split(data, "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if headerIndent >= 0 {
			if strings.TrimSpace(line) == "" || indent > headerIndent {
				content[i+1] = true
				continue
			}
			headerIndent = -1
		}
		if headerRe.MatchString(line) {
			headerIndent = indent
		}
	}

	return content
}
