package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/require"
)

// skipReasonMaxLines は、1 つの Skip 呼び出しを追う行数の上限です（gofmt の折り返しを吸収する）。
const skipReasonMaxLines = 8

var (
	// skipCallRe は、テスト内の Skip 呼び出し（Skip / Skipf）を検出する。
	skipCallRe = regexp.MustCompile(`\bSkipf?\(`)
	// coveringTestRe は、skip 理由が別のテスト関数を名指ししていることを検出する。
	// Go のテスト関数は Test の直後が大文字か `_` になるため、この形で「他テスト参照」を捕捉できる。
	coveringTestRe = regexp.MustCompile(`Test[A-Z_]`)
)

// TestSkipReasonDoesNotNameCoveringTest は、Skip の理由が他のテストを名指ししていないことを機械検証する。
// Skip が許されるのは対象が検証不可能であるために到達できない場合のみで、理由には「なぜ検証不可能か」を書く。
// 「他のテストでカバー済み」を理由にした skip はテストを別テストの実装へ依存させ、カバー元が縮小・削除されても
// 無言で green のまま残るため、1:1 マッピングを名前だけの殻にする。
func TestSkipReasonDoesNotNameCoveringTest(t *testing.T) {
	t.Parallel()

	var violations []string
	for _, root := range moduleSubdirs(t, "internal", "pkg") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := pkgfs.OS{}.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			violations = append(violations, collectSkipReasonViolations(path, strings.Split(string(src), "\n"))...)
			return nil
		})
		require.NoError(t, err)
	}

	sort.Strings(violations)
	for _, v := range violations {
		t.Log("他テストを名指しした skip: " + v)
	}

	require.Empty(t, violations,
		"skip の理由が他のテストを名指ししている。テスト可能な対象は実テストを書くこと。"+
			"検証不可能であるために到達できない場合に限り、理由に「なぜ検証不可能か」を書いて skip できる。")
}

// collectSkipReasonViolations は、他テストを名指しした Skip 呼び出しを違反として列挙する。
func collectSkipReasonViolations(file string, lines []string) []string {
	var violations []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		loc := skipCallRe.FindStringIndex(line)
		if loc == nil {
			continue
		}
		if coveringTestRe.MatchString(skipCallText(lines, i, loc[1])) {
			violations = append(violations, file+":"+strconv.Itoa(i+1)+": "+trimmed)
		}
	}
	return violations
}

// skipCallText は、Skip 呼び出しの引数テキストを返す。
// gofmt は長い引数を複数行へ折り返すため、閉じ括弧で終わる行までを連結する。
func skipCallText(lines []string, declIdx, from int) string {
	var text strings.Builder
	text.WriteString(lines[declIdx][from:])
	if strings.HasSuffix(strings.TrimSpace(lines[declIdx]), ")") {
		return text.String()
	}
	for j := declIdx + 1; j < len(lines) && j-declIdx <= skipReasonMaxLines; j++ {
		text.WriteString(lines[j])
		if strings.HasSuffix(strings.TrimSpace(lines[j]), ")") {
			break
		}
	}
	return text.String()
}
