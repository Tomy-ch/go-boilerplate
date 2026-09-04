// Package main は pnpm の minimumReleaseAgeExclude に期限を強制する検査ツール。
//
// 兄弟の go-cooldown / tool-cooldown とは守る対象が違う。あちらは窓そのものを強制する本体で、
// go.mod や mise.toml には解決時に窓を効かせる機構が無いために存在する。pnpm はこの repo で
// 唯一、resolver 自身が窓を強制する生態系で、しかも解決時だけでなく install のたびに lockfile
// 全体を照らし直す（--frozen-lockfile の再生も含む）。窓の番人はもう居る。
//
// 番人が居ないのは逃げ道のほうにある。minimumReleaseAgeExclude は特定バージョンだけ窓を外す
// 例外で、追加時は tracked かつ CODEOWNERS 配下のファイルへ書くのでレビューに乗るが、そこから
// 先を見る仕組みが無い。期限は行コメントの散文にしか無く、機械は読んでいなかった。結果として
// 一時的な例外と恒久的な allowlist が区別できなくなる。docs/design/security.md が
// 「A bypass is dated debt」と述べている dated を、pnpm でも機械が読めるようにするのがこの
// ツールの仕事にあたる。
//
// 期限は pnpm-workspace.yaml が変わらなくても訪れるので、pull request の trigger だけでは
// 取りこぼす。兄弟と同じくスケジュール実行にも載せる必要がある。
package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	workspaceFile = "pnpm-workspace.yaml"
	excludeKey    = "minimumReleaseAgeExclude:"

	// maxExclusionMonths は例外の期限に許す最大の先送り幅。上限を置くのは、期限を遠い未来へ
	// 置くだけで恒久 allowlist と同じ状態を作れてしまうため（go-cooldown の maxBypassMonths と同じ）。
	maxExclusionMonths = 3
)

var (
	// errViolations は、解消すべき違反が残っていることを表すエラー。
	errViolations = xerrors.New("pnpm cooldown exclusion violations")

	// entryRe は `  - <spec> # <メタ>` の 1 エントリを読む。メタが無い行も捕らえて違反にするため、
	// コメント部分は省略可能にしてある。
	entryRe = regexp.MustCompile(`^\s+-\s+(\S+)\s*(?:#\s*(.*))?$`)
	// specRe は name@version を分ける。@scope/name@version も 1 つ目の @ を名前側に残す。
	specRe = regexp.MustCompile(`^(@?[^@]+)@([^@]+)$`)
	// expiresRe / issueRe はメタから機械可読な 2 項目を拾う。
	expiresRe = regexp.MustCompile(`\bexpires:\s*(\d{4}-\d{2}-\d{2})\b`)
	issueRe   = regexp.MustCompile(`\bissue:\s*#?(\d+)\b`)
	// metaStripRe は理由を取り出すために、機械可読な 2 項目を落とす。
	metaStripRe = regexp.MustCompile(`\b(?:expires:\s*\d{4}-\d{2}-\d{2}|issue:\s*#?\d+)\b`)
)

// exclusion は minimumReleaseAgeExclude の 1 エントリ。
type exclusion struct {
	file    string
	line    int
	spec    string // 宣言に書かれたまま（name@version）
	name    string
	version string
	expires time.Time
	issue   int
	reason  string
	// malformed は、機械可読な形式を満たさなかったことを表す。満たさない限り期限は判定できない。
	malformed string
}

// main は 1:1 テスト規約の対象外で分岐を検査できないため、判断は run に置きます。
func main() {
	log.SetFlags(0)

	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ %v", err)
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if err := run(root, os.Stdout, today); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

// run は root 配下の pnpm-workspace.yaml を全て検査し、違反があればエラーを返します。
func run(root string, out io.Writer, today time.Time) error {
	files, err := findWorkspaces(root)
	if err != nil {
		return err
	}

	var all []exclusion
	for _, f := range files {
		got, err := parseWorkspace(root, f)
		if err != nil {
			return err
		}
		all = append(all, got...)
	}

	violations := validate(root, all, today)

	_, _ = fmt.Fprintf(out, "ℹ️ pnpm-cooldown: workspace %d 件 / 例外 %d 件 / 上限 %d ヶ月\n",
		len(files), len(all), maxExclusionMonths)

	if len(violations) == 0 {
		_, _ = fmt.Fprintf(out, "✅ pnpm-cooldown: 違反なし\n")
		return nil
	}

	for _, v := range violations {
		_, _ = fmt.Fprintf(out, "❌ %s\n", v)
	}

	return xerrors.Wrap(errViolations, strconv.Itoa(len(violations))+" 件")
}

// findWorkspaces は root 配下の pnpm-workspace.yaml を集めます。node_modules と vendor は
// 依存が持ち込んだ他人の宣言なので除外します。
func findWorkspaces(root string) ([]string, error) {
	var out []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "vendor", ".git":
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != workspaceFile {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "walk")
	}

	sort.Strings(out)

	return out, nil
}

// parseWorkspace は 1 つの pnpm-workspace.yaml から例外エントリを読み出します。
func parseWorkspace(root, rel string) ([]exclusion, error) {
	f, err := os.Open(filepath.Join(root, rel)) //nolint:gosec // 走査で見つけた repo 内の宣言ファイル
	if err != nil {
		return nil, xerrors.Wrap(err, rel)
	}
	defer func() { _ = f.Close() }()

	var (
		out     []exclusion
		inBlock bool
		lineNo  int
	)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lineNo++
		line := sc.Text()

		if after, ok := strings.CutPrefix(line, excludeKey); ok {
			// `minimumReleaseAgeExclude: []` は例外ゼロの明示なので、ブロックには入らない。
			inBlock = strings.TrimSpace(after) == ""
			continue
		}
		if !inBlock {
			continue
		}
		// インデントされたリスト項目とコメント行だけがブロックの続き。それ以外で抜ける。
		if trimmed := strings.TrimSpace(line); trimmed == "" || !strings.HasPrefix(line, " ") {
			inBlock = false
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			inBlock = false
			continue
		}
		out = append(out, newExclusion(rel, lineNo, m[1], m[2]))
	}

	if err := sc.Err(); err != nil {
		return nil, xerrors.Wrap(err, rel)
	}

	return out, nil
}

// newExclusion は 1 行分を解釈します。機械可読な形式を満たさない場合は malformed に理由を残し、
// 期限の判定は行いません（読めない期限を「切れていない」と扱わないため）。
func newExclusion(file string, line int, spec, meta string) exclusion {
	e := exclusion{file: file, line: line, spec: spec}

	sm := specRe.FindStringSubmatch(spec)
	if sm == nil {
		e.malformed = "パッケージ名とバージョンが name@version の形になっていません"
		return e
	}
	e.name, e.version = sm[1], sm[2]

	em := expiresRe.FindStringSubmatch(meta)
	if em == nil {
		e.malformed = "コメントに expires: YYYY-MM-DD がありません"
		return e
	}

	expires, err := time.Parse(time.DateOnly, em[1])
	if err != nil {
		e.malformed = "expires が日付として読めません: " + em[1]
		return e
	}
	e.expires = expires

	im := issueRe.FindStringSubmatch(meta)
	if im == nil {
		e.malformed = "コメントに issue: <N> がありません"
		return e
	}

	issue, err := strconv.Atoi(im[1])
	if err != nil {
		e.malformed = "issue が数値として読めません: " + im[1]
		return e
	}
	e.issue = issue

	e.reason = strings.TrimSpace(metaStripRe.ReplaceAllString(meta, ""))
	if e.reason == "" {
		e.malformed = "理由が書かれていません"
	}

	return e
}

// validate は例外エントリ自身の規約違反を集めます。期限切れを失敗にするのは、外したまま
// 放置された例外が恒久 allowlist と区別できなくなるためです。
func validate(root string, all []exclusion, today time.Time) []string {
	limit := today.AddDate(0, maxExclusionMonths, 0)

	var violations []string
	for _, e := range all {
		at := fmt.Sprintf("%s:%d %s", e.file, e.line, e.spec)

		switch {
		case e.malformed != "":
			violations = append(violations, fmt.Sprintf(
				"%s %s。形式は `- <name>@<version> # expires: YYYY-MM-DD issue: <N> <理由>` です",
				at, e.malformed,
			))
		case e.expires.Before(today):
			violations = append(violations, fmt.Sprintf(
				"%s の期限 %s が切れています。窓明けの版へ解決し直してエントリを消すか、期限を延ばす判断を #%d で記録してください",
				at, e.expires.Format(time.DateOnly), e.issue,
			))
		case e.expires.After(limit):
			violations = append(violations, fmt.Sprintf(
				"%s の期限 %s が上限（%s から %d ヶ月 = %s）を越えています。恒久 allowlist にしないための上限です",
				at, e.expires.Format(time.DateOnly),
				today.Format(time.DateOnly), maxExclusionMonths, limit.Format(time.DateOnly),
			))
		default:
			if !resolvedInLock(root, e) {
				violations = append(violations, at+" は同じディレクトリの pnpm-lock.yaml が解決していません。不要になったエントリは消してください")
			}
		}
	}

	return violations
}

// resolvedInLock は、その版を lockfile が実際に解決しているかを返します。解決していない例外は
// 何も守っておらず、消し忘れの残骸にあたります。lockfile が読めない場合は判定を諦めて true を
// 返します（読めないことを例外側の違反として報告すると、原因の切り分けができなくなるため）。
func resolvedInLock(root string, e exclusion) bool {
	lock := filepath.Join(root, filepath.Dir(e.file), "pnpm-lock.yaml")

	f, err := os.Open(lock) //nolint:gosec // 宣言ファイルと同じディレクトリの lockfile
	if err != nil {
		return true
	}
	defer func() { _ = f.Close() }()

	want := "  " + e.spec + ":"

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if sc.Text() == want {
			return true
		}
	}

	return sc.Err() != nil
}
