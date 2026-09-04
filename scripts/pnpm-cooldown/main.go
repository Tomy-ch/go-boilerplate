// Package main は pnpm の minimumReleaseAgeExclude に期限を強制する検査ツール。
//
// 兄弟の go-cooldown / tool-cooldown とは守る対象が違う。あちらは窓そのものを強制する本体で、
// go.mod や mise.toml には解決時に窓を効かせる機構が無いために存在する。pnpm はこの repo で
// 唯一、resolver 自身が窓を強制する生態系で、しかも解決時だけでなく install のたびに lockfile
// 全体を照らし直す（--frozen-lockfile の再生も含む）。窓の番人はもう居る。
//
// 番人が居ないのは逃げ道のほうにある。minimumReleaseAgeExclude は特定バージョンだけ窓を外す
// 例外で、追加時は tracked かつ CODEOWNERS 配下のファイルへ書くのでレビューに乗るが、そこから
// 先を見る仕組みが無かった。docs/design/security.md が「A bypass is dated debt」と述べている
// dated を、pnpm でも機械が読めるようにするのがこのツールの仕事にあたる。
//
// 期限の置き場は宣言のコメントではなく .github/pnpm-cooldown-bypass.toml。兄弟 2 つと同じ形に
// したのは、期限が pnpm にとって意味を持たない情報で、pnpm が読む宣言に相乗りさせる必然性が
// 無いため。構造化された TOML を読むことで、YAML のコメントを機械可読な宣言として扱うことに
// 伴う取りこぼしも消える。
//
// 宣言側は yaml.Node として読む。行走査でシーケンスを拾うと、YAML として妥当で pnpm も honor
// する書き方——キーと同じ桁のブロックシーケンス、フロー形式 `[a@1]`、コロン前の空白——を
// 取りこぼし、期限の無い例外が「例外ゼロ」として通る。同じ機構は scripts/actions-shellcheck が
// 先に使っている。
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

	"gopkg.in/yaml.v3"

	"go-boilerplate/pkg/xerrors"
)

const (
	workspaceFile = "pnpm-workspace.yaml"
	lockFile      = "pnpm-lock.yaml"
	excludeKey    = "minimumReleaseAgeExclude"
	bypassFile    = ".github/pnpm-cooldown-bypass.toml" //nolint:gosec // 資格情報ではなくバイパス lockfile のパス

	// maxBypassMonths は例外の期限に許す最大の先送り幅。
	maxBypassMonths = 3
)

var (
	// errViolations は、解消すべき違反が残っていることを表すエラー。
	errViolations = xerrors.New("pnpm cooldown exclusion violations")
	// errBypassInvalidLine は、バイパス lockfile に解釈できない行があったことを表すエラー。
	errBypassInvalidLine = xerrors.New("unreadable line in the bypass lockfile")
	// errBypassDuplicateKey は、バイパス lockfile に同じキーが 2 度現れたことを表すエラー。
	errBypassDuplicateKey = xerrors.New("duplicate key in the bypass lockfile")

	// bypassLineRe は `"key" = { expires = ..., issue = ..., reason = "..." }` を読む。
	// 兄弟 2 つと同じ形式にしてある。
	bypassLineRe = regexp.MustCompile(
		`^"([^"]+)"\s*=\s*\{\s*expires\s*=\s*(\d{4}-\d{2}-\d{2})\s*,\s*issue\s*=\s*(\d+)\s*,\s*reason\s*=\s*"([^"]*)"\s*\}$`,
	)
	// lockKeyRe は lockfile のパッケージキーを読む。引用符を剥がすのは、実物が
	// `'@scope/name@1.0.0':` の形で現れるため。
	lockKeyRe = regexp.MustCompile(`^ {2}(?:'([^']+)'|"([^"]+)"|([^'"\s][^:]*?))\s*:\s*$`)
	// peerSuffixRe は解決キー末尾の peer 指定を落とす（`name@1.0.0(peer@2.0.0)` → `name@1.0.0`）。
	peerSuffixRe = regexp.MustCompile(`(\([^()]*\))+$`)
)

// exclusion は minimumReleaseAgeExclude の 1 エントリ。
type exclusion struct {
	file string // repo ルートからの相対パス
	line int
	spec string // <name>@<version>
}

// bypass は例外 1 件に与えた期限。
type bypass struct {
	line    int
	expires time.Time
	issue   int
	reason  string
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

// run は root 配下の pnpm-workspace.yaml とバイパス lockfile を突き合わせ、違反があれば
// エラーを返します。
func run(root string, out io.Writer, today time.Time) error {
	files, err := findWorkspaces(root)
	if err != nil {
		return err
	}

	var exclusions []exclusion

	for _, f := range files {
		got, parseErr := parseWorkspace(root, f)
		if parseErr != nil {
			return parseErr
		}

		exclusions = append(exclusions, got...)
	}

	bypasses, err := parseBypasses(root)
	if err != nil {
		return err
	}

	violations := validate(root, exclusions, bypasses, today)

	_, _ = fmt.Fprintf(out, "ℹ️ pnpm-cooldown: workspace %d 件 / 例外 %d 件 / バイパス %d 件 / 上限 %d ヶ月\n",
		len(files), len(exclusions), len(bypasses), maxBypassMonths)

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

// parseWorkspace は 1 つの pnpm-workspace.yaml から例外エントリを読み出します。行走査ではなく
// ノードとして読むのは、YAML が同じ意味をいくつもの書き方で表せるためです。
func parseWorkspace(root, rel string) ([]exclusion, error) {
	raw, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // 走査で見つけた repo 内の宣言ファイル
	if err != nil {
		return nil, xerrors.Wrap(err, rel)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, xerrors.Wrap(err, rel)
	}

	seq := findExcludeSequence(&doc)
	if seq == nil {
		return nil, nil
	}

	out := make([]exclusion, 0, len(seq.Content))
	for _, item := range seq.Content {
		out = append(out, exclusion{file: rel, line: item.Line, spec: item.Value})
	}

	return out, nil
}

// findExcludeSequence は文書直下のマッピングから minimumReleaseAgeExclude のシーケンスを
// 探します。見つからない、あるいはシーケンスでない場合は nil を返します。
func findExcludeSequence(doc *yaml.Node) *yaml.Node {
	if len(doc.Content) == 0 {
		return nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}

	// マッピングの Content はキーと値が交互に並ぶ。
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != excludeKey {
			continue
		}

		if v := root.Content[i+1]; v.Kind == yaml.SequenceNode {
			return v
		}

		return nil
	}

	return nil
}

// parseBypasses はバイパス lockfile を読みます。解釈できない行と重複キーはエラーにします。
// 読み飛ばすと、書き損じた期限が「期限なし」ではなく「無検査」になってしまうためです。
func parseBypasses(root string) (map[string]bypass, error) {
	f, err := os.Open(filepath.Join(root, bypassFile)) //nolint:gosec // repo 内の固定パス
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bypass{}, nil
		}

		return nil, xerrors.Wrap(err, bypassFile)
	}
	defer func() { _ = f.Close() }()

	out := map[string]bypass{}
	lineNo := 0

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lineNo++

		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, key, err := parseBypassLine(line, lineNo)
		if err != nil {
			return nil, err
		}

		if _, dup := out[key]; dup {
			return nil, xerrors.Wrap(errBypassDuplicateKey, fmt.Sprintf("%s:%d %q", bypassFile, lineNo, key))
		}

		out[key] = entry
	}

	if err := sc.Err(); err != nil {
		return nil, xerrors.Wrap(err, bypassFile)
	}

	return out, nil
}

// parseBypassLine は 1 行を 1 エントリへ解釈します。
func parseBypassLine(line string, lineNo int) (bypass, string, error) {
	m := bypassLineRe.FindStringSubmatch(line)
	if m == nil {
		return bypass{}, "", xerrors.Wrap(errBypassInvalidLine, fmt.Sprintf(
			`%s:%d %q（形式は "<name>@<version>" = { expires = YYYY-MM-DD, issue = <N>, reason = "..." }）`,
			bypassFile, lineNo, line,
		))
	}

	expires, err := time.Parse(time.DateOnly, m[2])
	if err != nil {
		return bypass{}, "", xerrors.Wrap(err, fmt.Sprintf("%s:%d の expires", bypassFile, lineNo))
	}

	issue, err := strconv.Atoi(m[3])
	if err != nil {
		return bypass{}, "", xerrors.Wrap(err, fmt.Sprintf("%s:%d の issue", bypassFile, lineNo))
	}

	return bypass{line: lineNo, expires: expires, issue: issue, reason: m[4]}, m[1], nil
}

// validate は例外とバイパスを突き合わせ、違反を集めます。期限切れを失敗にするのは、外したまま
// 放置された例外が恒久 allowlist と区別できなくなるためです。
func validate(root string, exclusions []exclusion, bypasses map[string]bypass, today time.Time) []string {
	limit := today.AddDate(0, maxBypassMonths, 0)
	seen := map[string]struct{}{}

	var violations []string

	for _, e := range exclusions {
		seen[e.spec] = struct{}{}
		violations = append(violations, checkExclusion(root, e, bypasses, today, limit)...)
	}

	for _, key := range sortedKeys(bypasses) {
		if _, ok := seen[key]; ok {
			continue
		}

		violations = append(violations, fmt.Sprintf(
			"%s:%d %s はどの %s の %s にもありません。不要になったエントリは消してください",
			bypassFile, bypasses[key].line, key, workspaceFile, excludeKey,
		))
	}

	return violations
}

// checkExclusion は例外 1 件分の違反を返します。期限の違反で打ち切らず残骸も併せて見るのは、
// 片方ずつ直させると往復が増え、期限を延ばした次の実行で初めて残骸だと分かるためです。
func checkExclusion(
	root string, e exclusion, bypasses map[string]bypass, today, limit time.Time,
) []string {
	at := fmt.Sprintf("%s:%d %s", e.file, e.line, e.spec)

	b, ok := bypasses[e.spec]
	if !ok {
		return []string{fmt.Sprintf(
			"%s に期限がありません。%s へ "+`"%s" = { expires = YYYY-MM-DD, issue = <N>, reason = "..." }`+" を足してください",
			at, bypassFile, e.spec,
		)}
	}

	var out []string

	switch {
	case b.expires.Before(today):
		out = append(out, fmt.Sprintf(
			"%s の期限 %s（%s:%d）が切れています。窓明けの版へ解決し直して例外を消すか、期限を延ばす判断を #%d で記録してください",
			at, b.expires.Format(time.DateOnly), bypassFile, b.line, b.issue,
		))
	case b.expires.After(limit):
		out = append(out, fmt.Sprintf(
			"%s の期限 %s（%s:%d）が上限（%s から %d ヶ月 = %s）を越えています。恒久 allowlist にしないための上限です",
			at, b.expires.Format(time.DateOnly), bypassFile, b.line,
			today.Format(time.DateOnly), maxBypassMonths, limit.Format(time.DateOnly),
		))
	}

	if msg := checkResolved(root, e); msg != "" {
		out = append(out, msg)
	}

	return out
}

// checkResolved は、その版を lockfile が実際に解決しているかを検査します。違反があればその
// 文言を、無ければ空文字列を返します。lockfile が読めない場合も違反として報告します。読めない
// ことを「解決済み」に倒すと、lockfile を欠落させるだけで残骸の検知を無効化できるためです。
func checkResolved(root string, e exclusion) string {
	lock := filepath.Join(root, filepath.Dir(e.file), lockFile)

	specs, err := lockedSpecs(lock)
	if err != nil {
		return fmt.Sprintf("%s:%d %s の照合先 %s が読めません: %v", e.file, e.line, e.spec, lockFile, err)
	}

	if _, ok := specs[e.spec]; ok {
		return ""
	}

	return fmt.Sprintf(
		"%s:%d %s は同じディレクトリの %s が解決していません。不要になった例外は消してください",
		e.file, e.line, e.spec, lockFile,
	)
}

// lockedSpecs は lockfile が解決している <name>@<version> の集合を返します。引用符と peer
// サフィックスを剥がすのは、実物が `'@scope/name@1.0.0':` や `name@1.0.0(peer@2.0.0):` の形で
// 現れるためで、素の完全一致では scoped パッケージが常に未解決に見えてしまいます。
func lockedSpecs(path string) (map[string]struct{}, error) {
	f, err := os.Open(path) //nolint:gosec // 宣言ファイルと同じディレクトリの lockfile
	if err != nil {
		return nil, xerrors.Wrap(err, path)
	}
	defer func() { _ = f.Close() }()

	out := map[string]struct{}{}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), bufio.MaxScanTokenSize)

	for sc.Scan() {
		m := lockKeyRe.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}

		key := peerSuffixRe.ReplaceAllString(m[1]+m[2]+m[3], "") // 引用符の種類ごとに 1 つだけ埋まる
		if strings.Contains(key, "@") {
			out[key] = struct{}{}
		}
	}

	if err := sc.Err(); err != nil {
		return nil, xerrors.Wrap(err, path)
	}

	return out, nil
}

// sortedKeys は報告の順序を安定させます。
func sortedKeys(m map[string]bypass) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
