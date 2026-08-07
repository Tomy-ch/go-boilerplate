// Package main は mise.toml が宣言するツールのバージョンが供給網 cooldown を満たしているかを
// 検査するツール。
//
//	gate:  base からの差分で追加 / 更新されたツールのうち、公開から窓日数未満のものを違反として
//	       報告し、非ゼロで終了する。
//	audit: mise.toml 全件を棚卸しし、窓内のものを報告する。常に 0 で終了する。
//
// **この規約は既に人手で運用されていた。** mise.toml のコメントが「最新は v1.26.0 だが公開 2 日で
// クールダウン 14 日に満たないため 1 つ前を採る」と記録しており、窓の値も backend ごとに決まって
// いる。機械化されていなかっただけで、ここで新しい方針を導入するわけではない。
//
// **窓は backend の性質で 2 段。** GitHub リリースに紐づくもの（aqua / ubi / github）は 14 日で、
// tag が別 commit へ付け替えられる形の侵害があり得るぶん pin-actions / pin-images と同じ窓を採る。
// パッケージレジストリに紐づくもの（go / npm / pipx）は公開が immutable なので 7 日で、
// npm-cooldown / go-cooldown と揃う。
//
// **言語ランタイム（core backend）は対象外。** go / node / python 本体が汚染される事態は、
// 供給網の一部ではなく言語エコシステムの信頼性そのものが崩れたときであり、冷却期間で守れる
// ものが無い。影響範囲も基盤全体に及ぶ。したがってこれは検査の穴ではなく、明示的に受容した
// リスクである。ランタイムには LTS を待つという別軸の方針があり、そちらが担う。
//
// **短縮名の backend は mise registry に解決させる。** どの backend で解決されるかを決めているのは
// mise 自身なので、対応表を持つと mise の更新で静かにずれる。
//
// **緊急のバイパスは .github/mise-cooldown-bypass.toml が受ける。** エントリは期限を必ず持ち、
// 期限切れ・期限過長・対象不在は gate / audit のどちらでも失敗になる。期限は mise.toml が
// 変わらなくても訪れるので、このツールはスケジュール実行にも載せる必要がある。
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	miseFile   = "mise.toml"
	bypassFile = ".github/mise-cooldown-bypass.toml" //nolint:gosec // 資格情報ではなくバイパス lockfile のパス

	// releaseWindowDays は GitHub リリースに紐づく backend の窓。
	releaseWindowDays = 14
	// registryWindowDays はパッケージレジストリに紐づく backend の窓。
	registryWindowDays = 7

	maxBypassMonths = 3 // バイパスの期限に許す最大の先送り幅

	fetchTimeout = 30 * time.Second
	miseTimeout  = 30 * time.Second
	gitTimeout   = 30 * time.Second
	fetchWorkers = 4 // GitHub API のレート制限に配慮して go-cooldown より絞る
	minFenceLen  = 3 // CommonMark のフェンス下限
	hoursPerDay  = 24
	summaryPerm  = 0o644
	outputPerm   = 0o600

	githubAPI   = "https://api.github.com/repos/"
	goProxyBase = "https://proxy.golang.org/"
	npmBase     = "https://registry.npmjs.org/"
	pypiBase    = "https://pypi.org/pypi/"
)

var (
	// errUsage は、サブコマンドやフラグの与え方が誤っていることを表すエラー。
	errUsage = xerrors.New("invalid usage")
	// errBlocking は、gate を通せない違反が残っていることを表すエラー。
	errBlocking = xerrors.New("blocking violations")
	// errUnsupportedBackend は、公開時刻の取得経路を持たない backend を指すエラー。
	errUnsupportedBackend = xerrors.New("no publish-time source for this backend")
	// errNotFound は、上流がそのバージョンを知らない場合のエラー。
	errNotFound = xerrors.New("version not found upstream")
	// errUpstreamStatus は、上流が想定外のステータスを返した場合のエラー。
	errUpstreamStatus = xerrors.New("unexpected upstream status")
	// errBypassInvalidLine は、バイパス lockfile に解釈できない行があった場合のエラー。
	errBypassInvalidLine = xerrors.New("invalid bypass line")
	// errBypassDuplicateKey は、バイパス lockfile にキーの重複があった場合のエラー。
	errBypassDuplicateKey = xerrors.New("duplicate bypass key")

	// toolLineRe は `[tools]` の 1 行を key と version へ割る。key は裸でも引用符付きでもよい。
	toolLineRe = regexp.MustCompile(`^\s*(?:"([^"]+)"|([A-Za-z0-9_.\-]+))\s*=\s*"([^"]+)"\s*$`)
	// sectionRe は TOML のセクション見出し。
	sectionRe = regexp.MustCompile(`^\s*\[([^\]]+)\]\s*$`)
	// extrasRe は pipx の extras 表記（`graphifyy[sql]` の `[sql]`）。
	extrasRe = regexp.MustCompile(`\[[^\]]*\]$`)
	// bypassLineRe は `"key@version" = { expires = ..., issue = ..., reason = "..." }` を読む。
	bypassLineRe = regexp.MustCompile(
		`^"([^"]+)"\s*=\s*\{\s*expires\s*=\s*(\d{4}-\d{2}-\d{2})\s*,\s*issue\s*=\s*(\d+)\s*,\s*reason\s*=\s*"([^"]*)"\s*\}$`)
)

// tool は mise.toml `[tools]` の 1 行。backend は解決後の値で、短縮名なら mise registry が決める。
type tool struct {
	key     string // mise.toml に書かれたキーそのもの
	version string
	backend string // 例: aqua:owner/repo, go:module, npm:pkg, pipx:pkg, core:go
}

// bypass は cooldown を明示的に外す 1 エントリ。
type bypass struct {
	expires time.Time
	issue   int
	reason  string
	line    int
}

// finding は窓を満たさないツール 1 件。
type finding struct {
	tool      tool
	published time.Time
	ageDays   int
	window    int
}

// options はサブコマンドに続けて与えられたフラグ。
type options struct {
	base       string
	summaryOut string
	github     bool
}

func (t tool) id() string { return t.key + "@" + t.version }

// main はエラーを終了コードへ変換するだけに留め、判断は run が持ちます。
// main は 1:1 の対象外でテストを書けないため、ここに分岐を置くと検査されない
// コードがそのぶん増える。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:], &http.Client{Timeout: fetchTimeout}, time.Now().UTC()); err != nil {
		log.Fatalf("%v", err)
	}
}

// run は mise.toml の宣言を cooldown に照らして報告します。gate で違反が残った場合はエラーを
// 返し、終了コードへの変換は呼び出し側に委ねます。
// client は上流への問い合わせ手段、now は窓と期限の基準時刻で、差し替えられるよう引数で受けます。
func run(args []string, client *http.Client, now time.Time) error {
	sub, opt, err := parseArgs(args)
	if err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	// 期限の比較は暦日で行う。時刻を残すと 3 ヶ月の境界が実行時刻とタイムゾーンで動く。
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	raw, err := os.ReadFile(miseFile)
	if err != nil {
		return xerrors.Wrap(err, "❌ read "+miseFile)
	}
	declared, err := parseTools(raw)
	if err != nil {
		return xerrors.Wrap(err, "❌ "+miseFile)
	}

	bypasses, err := readBypasses(bypassFile)
	if err != nil {
		return xerrors.Wrap(err, "❌ "+bypassFile)
	}
	policyViolations, invalidBypasses := validateBypasses(bypasses, declared, today)

	targets := declared
	if sub == "gate" {
		targets, err = added(opt.base, declared)
		if err != nil {
			return xerrors.Wrap(err, "❌ base との差分")
		}
	}

	ctx := context.Background()
	resolved, skipped, err := resolveBackends(ctx, targets)
	if err != nil {
		return xerrors.Wrap(err, "❌ backend の解決")
	}

	findings, unresolved := inspect(ctx, client, resolved, now)

	blocking := report(sub, findings, unresolved, skipped, policyViolations, bypasses, invalidBypasses, resolved, opt.github)

	if opt.summaryOut != "" {
		body := summary(sub, findings, unresolved, skipped, policyViolations, bypasses, invalidBypasses, resolved)
		if writeErr := os.WriteFile(opt.summaryOut, []byte(body), summaryPerm); writeErr != nil {
			return xerrors.Wrap(writeErr, "❌ write summary")
		}
	}
	if out := os.Getenv("GITHUB_OUTPUT"); out != "" {
		if writeErr := appendOutput(out, len(findings), len(resolved), blocking); writeErr != nil {
			return xerrors.Wrap(writeErr, "❌ write GITHUB_OUTPUT")
		}
	}

	if blocking > 0 {
		return xerrors.Wrap(errBlocking, fmt.Sprintf("❌ mise-cooldown %s: %d 件の違反が残っています", sub, blocking))
	}

	return nil
}

// parseArgs はサブコマンドとフラグを解釈します。ヘルプ要求は flag.ErrHelp を含むエラーとして
// 返し、失敗として扱うかどうかの判断は呼び出し側に委ねます。
func parseArgs(args []string) (string, options, error) {
	if len(args) == 0 {
		return "", options{}, xerrors.Wrap(errUsage,
			"❌ usage: mise-cooldown <gate|audit> [--base=<git-ref>] [--summary-out=<path>] [--github]")
	}

	sub := args[0]
	if sub != "gate" && sub != "audit" {
		return "", options{}, xerrors.Wrap(errUsage, fmt.Sprintf("❌ 未知のサブコマンド %q（gate | audit）", sub))
	}

	fs := flag.NewFlagSet(sub, flag.ContinueOnError)
	base := fs.String("base", "", "gate: この git ref からの差分で追加 / 更新されたツールだけを対象にする")
	summaryOut := fs.String("summary-out", "", "Markdown サマリの出力先")
	github := fs.Bool("github", false, "GitHub Actions のアノテーションを出力する")
	if err := fs.Parse(args[1:]); err != nil {
		return "", options{}, xerrors.Wrap(err, "❌ フラグを解釈できません")
	}

	// gate は差分を取れないと何も検査しないまま通るため、比較対象の不在を先に落とす。
	if sub == "gate" && *base == "" {
		return "", options{}, xerrors.Wrap(errUsage, "❌ gate には --base が要ります（比較対象が無いと差分を取れません）")
	}

	return sub, options{base: *base, summaryOut: *summaryOut, github: *github}, nil
}

// parseTools は `[tools]` セクションの宣言だけを読む。`[settings]` の `pipx.uvx` や `[env]` の
// バージョン値をツールとして拾わないよう、セクションを見て範囲を限る。
func parseTools(content []byte) ([]tool, error) {
	var tools []tool
	section := ""
	sc := bufio.NewScanner(strings.NewReader(string(content)))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := sectionRe.FindStringSubmatch(trimmed); m != nil {
			section = m[1]
			continue
		}
		if section != "tools" {
			continue
		}
		m := toolLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		key := m[1]
		if key == "" {
			key = m[2]
		}
		tools = append(tools, tool{key: key, version: m[3]})
	}
	return tools, sc.Err()
}

// added は base 時点の mise.toml に無かった (key, version) の組を返す。
func added(base string, current []tool) ([]tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "git", "show", base+":"+miseFile).Output() //nolint:gosec // base は呼び出し側が与える git ref
	if err != nil {
		return nil, xerrors.Wrap(err, fmt.Sprintf("git show %s:%s（base を取得できていない可能性があります）", base, miseFile))
	}
	return addedFrom(out, current)
}

// addedFrom は base 時点の mise.toml の中身と現在の宣言を突き合わせる。base を読めなかった場合に
// 空の差分を返すと、gate は何も検査しないまま通る。解析の失敗はエラーとして表に出す。
func addedFrom(baseContent []byte, current []tool) ([]tool, error) {
	before, err := parseTools(baseContent)
	if err != nil {
		return nil, xerrors.Wrap(err, "base の "+miseFile)
	}
	return diffAdded(before, current), nil
}

// diffAdded は before に無い (key, version) の組を返す。
func diffAdded(before, current []tool) []tool {
	known := make(map[string]struct{}, len(before))
	for _, t := range before {
		known[t.id()] = struct{}{}
	}
	var fresh []tool
	for _, t := range current {
		if _, ok := known[t.id()]; !ok {
			fresh = append(fresh, t)
		}
	}
	return fresh
}

// resolveBackends は各ツールの backend を決め、対象と除外（core backend）へ振り分ける。
// 短縮名は mise registry の先頭候補を採る。mise 自身が選ぶのと同じ順序である。
func resolveBackends(ctx context.Context, tools []tool) ([]tool, []tool, error) {
	var targets, skipped []tool
	for _, t := range tools {
		if strings.Contains(t.key, ":") {
			t.backend = t.key
		} else {
			backend, resolveErr := miseRegistry(ctx, t.key)
			if resolveErr != nil {
				return nil, nil, resolveErr
			}
			t.backend = backend
		}
		if strings.HasPrefix(t.backend, "core:") {
			skipped = append(skipped, t)
			continue
		}
		targets = append(targets, t)
	}
	return targets, skipped, nil
}

func miseRegistry(ctx context.Context, name string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, miseTimeout)
	defer cancel()

	out, err := exec.CommandContext(cctx, "mise", "registry", name).Output() //nolint:gosec // name は mise.toml のキー
	if err != nil {
		return "", xerrors.Wrap(err, "mise registry "+name)
	}
	return firstBackend(out, name)
}

// firstBackend は `mise registry` の出力から先頭の候補だけを採る。mise 自身が選ぶのと同じ順序で、
// 出力全体を採ると backend が空白を含み、以降の種別判定がすべて空振りする。
func firstBackend(registryOutput []byte, name string) (string, error) {
	fields := strings.Fields(string(registryOutput))
	if len(fields) == 0 {
		return "", xerrors.Wrap(errUnsupportedBackend, name)
	}
	return fields[0], nil
}

// windowFor は backend の性質から窓を決める。
func windowFor(backend string) int {
	switch backendKind(backend) {
	case "github":
		return releaseWindowDays
	default:
		return registryWindowDays
	}
}

func backendKind(backend string) string {
	name, _, _ := strings.Cut(backend, ":")
	switch name {
	case "aqua", "ubi", "github":
		return "github"
	case "go":
		return "go"
	case "npm":
		return "npm"
	case "pipx", "uvx":
		return "pypi"
	default:
		return ""
	}
}

// inspect は対象ツールの公開時刻を引き、窓内のものと取得できなかったものを返す。
func inspect(ctx context.Context, client *http.Client, targets []tool, now time.Time) ([]finding, []tool) {
	var (
		mu         sync.Mutex
		findings   []finding
		unresolved []tool
		wg         sync.WaitGroup
	)
	sem := make(chan struct{}, fetchWorkers)

	for _, t := range targets {
		wg.Add(1)
		go func(t tool) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			at, err := publishedAt(ctx, client, t)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				unresolved = append(unresolved, t)
				return
			}
			window := windowFor(t.backend)
			if age := int(now.Sub(at).Hours() / hoursPerDay); age < window {
				findings = append(findings, finding{tool: t, published: at, ageDays: age, window: window})
			}
		}(t)
	}
	wg.Wait()

	sort.Slice(findings, func(i, j int) bool { return findings[i].ageDays < findings[j].ageDays })
	sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].id() < unresolved[j].id() })
	return findings, unresolved
}

// publishedAt は backend ごとの上流からそのバージョンの公開時刻を取る。
func publishedAt(ctx context.Context, client *http.Client, t tool) (time.Time, error) {
	_, ref, _ := strings.Cut(t.backend, ":")
	switch backendKind(t.backend) {
	case "github":
		return githubReleaseAt(ctx, client, ref, t.version)
	case "go":
		return goModuleAt(ctx, client, ref, t.version)
	case "npm":
		return npmPackageAt(ctx, client, ref, t.version)
	case "pypi":
		return pypiPackageAt(ctx, client, extrasRe.ReplaceAllString(ref, ""), t.version)
	default:
		return time.Time{}, xerrors.Wrap(errUnsupportedBackend, t.backend)
	}
}

func getJSON(ctx context.Context, client *http.Client, url string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return xerrors.Wrap(err, "new request")
	}
	// 未認証の GitHub API は 60 req/hour（IP 単位）で、このツールの 1 回の実行を賄えない。
	if strings.HasPrefix(url, githubAPI) {
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return xerrors.Wrap(err, "fetch "+url)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return xerrors.Wrap(errNotFound, url)
	}
	if resp.StatusCode != http.StatusOK {
		return xerrors.Wrap(errUpstreamStatus, fmt.Sprintf("%s: %d", url, resp.StatusCode))
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

// githubReleaseAt は Release の published_at を返す。タグに `v` を付ける流儀と付けない流儀が
// あるため、宣言された版そのものと `v` 付きの両方を試す。
func githubReleaseAt(ctx context.Context, client *http.Client, repo, version string) (time.Time, error) {
	var body struct {
		//nolint:tagliatelle // GitHub API の応答フィールド名
		PublishedAt time.Time `json:"published_at"`
	}
	var lastErr error
	for _, tag := range []string{"v" + version, version} {
		err := getJSON(ctx, client, githubAPI+repo+"/releases/tags/"+tag, &body)
		if err == nil {
			return body.PublishedAt, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// goModuleAt は module proxy の .info を返す。mise の go backend はパッケージパスを受けるが
// proxy が知るのはモジュールパスなので、解決するまで末尾要素を落としながら遡る。
func goModuleAt(ctx context.Context, client *http.Client, pkg, version string) (time.Time, error) {
	var body struct {
		//nolint:tagliatelle // module proxy の応答フィールド名
		Time time.Time `json:"Time"`
	}
	var lastErr error
	for path := pkg; strings.Contains(path, "/"); path = path[:strings.LastIndex(path, "/")] {
		err := getJSON(ctx, client, goProxyBase+escapeModulePath(path)+"/@v/v"+version+".info", &body)
		if err == nil {
			return body.Time, nil
		}
		lastErr = err
	}
	// pkg に `/` が無いとループが 1 度も回らない。ここで nil を返すと呼び出し側はゼロ値時刻を
	// 「公開から数十年経過」と読み、そのツールが cooldown を無条件で通過する。「問い合わせた結果
	// 見つからない」と「一度も問い合わせていない」は、公開時刻を得られていない点で同じ扱いにする。
	if lastErr == nil {
		return time.Time{}, xerrors.Wrap(errNotFound, pkg+"@"+version)
	}
	return time.Time{}, lastErr
}

// escapeModulePath は module path を proxy のパス表記へ変換する。proxy は大文字を `!` + 小文字で
// 受けるため、未エスケープのまま問い合わせると 404 になり「存在しないバージョン」に化ける。
func escapeModulePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func npmPackageAt(ctx context.Context, client *http.Client, pkg, version string) (time.Time, error) {
	var body struct {
		Time map[string]time.Time `json:"time"`
	}
	if err := getJSON(ctx, client, npmBase+pkg, &body); err != nil {
		return time.Time{}, err
	}
	at, ok := body.Time[version]
	if !ok {
		return time.Time{}, xerrors.Wrap(errNotFound, pkg+"@"+version)
	}
	return at, nil
}

func pypiPackageAt(ctx context.Context, client *http.Client, pkg, version string) (time.Time, error) {
	var body struct {
		Releases map[string][]struct {
			//nolint:tagliatelle // PyPI の応答フィールド名
			UploadTime time.Time `json:"upload_time_iso_8601"`
		} `json:"releases"`
	}
	if err := getJSON(ctx, client, pypiBase+pkg+"/json", &body); err != nil {
		return time.Time{}, err
	}
	files, ok := body.Releases[version]
	if !ok || len(files) == 0 {
		return time.Time{}, xerrors.Wrap(errNotFound, pkg+"@"+version)
	}
	return files[0].UploadTime, nil
}

// readBypasses はバイパス lockfile を key@version→bypass として読む。空行とコメント行以外で
// 解釈できない行、および既出キーの再定義はエラーにする。読み飛ばしや後勝ちの上書きは、その
// エントリが「存在しない」あるいは「行順で決まる」状態を警告なく作る。
func readBypasses(path string) (map[string]bypass, error) {
	f, err := os.Open(path) //nolint:gosec // path はリテラル
	if os.IsNotExist(err) {
		return map[string]bypass{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	out := map[string]bypass{}
	sc := bufio.NewScanner(f)
	for lineNo := 1; sc.Scan(); lineNo++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := bypassLineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, xerrors.Wrap(errBypassInvalidLine, fmt.Sprintf(
				`%d 行目: %q（形式は "<key>@<version>" = { expires = YYYY-MM-DD, issue = <N>, reason = "..." }）`, lineNo, line))
		}
		if _, dup := out[m[1]]; dup {
			return nil, xerrors.Wrap(errBypassDuplicateKey, fmt.Sprintf("%d 行目: %q", lineNo, m[1]))
		}
		expires, err := time.Parse(time.DateOnly, m[2])
		if err != nil {
			return nil, xerrors.Wrap(err, fmt.Sprintf("%d 行目の expires", lineNo))
		}
		issue, err := strconv.Atoi(m[3])
		if err != nil {
			return nil, xerrors.Wrap(err, fmt.Sprintf("%d 行目の issue", lineNo))
		}
		out[m[1]] = bypass{expires: expires, issue: issue, reason: m[4], line: lineNo}
	}
	return out, sc.Err()
}

// validateBypasses はバイパス自身の規約違反と、そのせいで無効になったキーを返す。期限切れを
// 失敗にするのは、外したまま放置されたバイパスが恒久 allowlist と区別できなくなるため。上限を
// 置くのは、期限を遠い未来へ置くだけで同じ状態を作れてしまうため。無効なバイパスは効力も失う。
func validateBypasses(bypasses map[string]bypass, declared []tool, today time.Time) ([]string, map[string]struct{}) {
	inMise := make(map[string]struct{}, len(declared))
	for _, t := range declared {
		inMise[t.id()] = struct{}{}
	}
	limit := today.AddDate(0, maxBypassMonths, 0)
	invalid := map[string]struct{}{}
	var violations []string

	for _, key := range sortedKeys(bypasses) {
		b := bypasses[key]
		switch {
		case b.expires.Before(today):
			invalid[key] = struct{}{}
			violations = append(violations, fmt.Sprintf(
				"%s:%d %s の期限 %s が切れています。窓明けの版へ更新してエントリを消すか、期限を延ばす判断を #%d で記録してください",
				bypassFile, b.line, key, b.expires.Format(time.DateOnly), b.issue))
		case b.expires.After(limit):
			invalid[key] = struct{}{}
			violations = append(violations, fmt.Sprintf(
				"%s:%d %s の期限 %s が上限（%s から %d ヶ月 = %s）を越えています。恒久 allowlist にしないための上限です",
				bypassFile, b.line, key, b.expires.Format(time.DateOnly),
				today.Format(time.DateOnly), maxBypassMonths, limit.Format(time.DateOnly)))
		default:
			if _, ok := inMise[key]; !ok {
				violations = append(violations, fmt.Sprintf(
					"%s:%d %s は %s に存在しません。不要になったエントリは消してください",
					bypassFile, b.line, key, miseFile))
			}
		}
	}
	return violations, invalid
}

func sortedKeys(m map[string]bypass) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// classify は finding を「バイパス済み」「gate を落とすもの」「報告に留めるもの」へ振り分ける。
func classify(sub string, findings []finding, bypasses map[string]bypass, invalid map[string]struct{}) ([]finding, []finding, []finding) {
	var bypassed, blocking, reported []finding
	for _, f := range findings {
		switch {
		case hasBypass(bypasses, invalid, f.tool):
			bypassed = append(bypassed, f)
		case sub == "gate":
			blocking = append(blocking, f)
		default:
			reported = append(reported, f)
		}
	}
	return bypassed, blocking, reported
}

func hasBypass(bypasses map[string]bypass, invalid map[string]struct{}, t tool) bool {
	if _, bad := invalid[t.id()]; bad {
		return false
	}
	_, ok := bypasses[t.id()]
	return ok
}

// report は結果を標準出力へ書き、終了コードを非ゼロにすべき件数を返す。audit は窓内の finding では
// 落ちない。ただしバイパス自身の規約違反だけは audit でも失敗させる。期限切れの回収がスケジュール
// 実行に懸かっているため。
func report(
	sub string, findings []finding, unresolved, skipped []tool,
	policyViolations []string, bypasses map[string]bypass, invalid map[string]struct{},
	targets []tool, github bool,
) int {
	//nolint:gosec // sub は gate / audit のいずれかに限定済み
	log.Printf("ℹ️ mise-cooldown %s: 対象 %d 件 / ランタイム除外 %d 件 / 窓 GitHub %d 日・レジストリ %d 日",
		sub, len(targets), len(skipped), releaseWindowDays, registryWindowDays)

	blockingCount := len(policyViolations)
	for _, v := range policyViolations {
		log.Printf("❌ %s", v)
		if github {
			log.Printf("::error file=%s::%s", bypassFile, v)
		}
	}

	bypassed, blocked, reported := classify(sub, findings, bypasses, invalid)

	for _, f := range bypassed {
		b := bypasses[f.tool.id()]
		log.Printf("🔓 %s は公開 %d 日ですが、#%d のバイパスで通します（期限 %s / %s）",
			f.tool.id(), f.ageDays, b.issue, b.expires.Format(time.DateOnly), b.reason)
	}

	blockingCount += len(blocked)
	for _, f := range blocked {
		msg := fmt.Sprintf("%s（%s）は公開 %d 日で cooldown %d 日を満たしていません。窓明けを待つか、"+
			"待てない根拠を %s へ期限付きで記録してください", f.tool.id(), f.tool.backend, f.ageDays, f.window, bypassFile)
		log.Printf("❌ %s", msg)
		if github {
			log.Printf("::error file=%s::%s", miseFile, msg)
		}
	}

	for _, f := range reported {
		log.Printf("⚠️ %s（%s）は公開 %d 日で cooldown %d 日を満たしていません", f.tool.id(), f.tool.backend, f.ageDays, f.window)
	}

	for _, t := range unresolved {
		// 取得できないものを黙って通すと、検査が「見た結果 OK」と「見ていない」を区別できなくなる。
		msg := fmt.Sprintf("%s（%s）の公開時刻を取得できませんでした", t.id(), t.backend)
		if sub == "gate" {
			blockingCount++
			log.Printf("❌ %s", msg)
			if github {
				log.Printf("::error file=%s::%s", miseFile, msg)
			}
			continue
		}
		log.Printf("⚠️ %s", msg)
	}

	if blockingCount == 0 {
		//nolint:gosec // sub は gate / audit のいずれかに限定済み
		log.Printf("✅ mise-cooldown %s: 違反なし（バイパス %d 件）", sub, len(bypasses))
	}
	return blockingCount
}

// fenceFor は text を包むのに足りるフェンスを返す。長さは text 中の最長バッククォート連 + 1。
func fenceFor(text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
			continue
		}
		run = 0
	}
	return strings.Repeat("`", max(minFenceLen, longest+1))
}

// fenced は見出しと、フェンスで包んだ本体を書く。値は mise.toml 由来で pull request が中身を
// 決めるため、見出しだけをテンプレート側に残して値はフェンスへ入れる。
func fenced(b *strings.Builder, heading string, lines []string) {
	body := strings.Join(lines, "\n")
	fence := fenceFor(body)
	fmt.Fprintf(b, "## %s (%d)\n\n%stext\n%s\n%s\n\n", heading, len(lines), fence, body, fence)
}

// summary は GITHUB_STEP_SUMMARY 用の Markdown を組む。
func summary(
	sub string, findings []finding, unresolved, skipped []tool,
	policyViolations []string, bypasses map[string]bypass, invalid map[string]struct{},
	targets []tool,
) string {
	var b strings.Builder
	_, blocked, reported := classify(sub, findings, bypasses, invalid)

	fmt.Fprintf(&b, "対象 %d 件 / ランタイム除外 %d 件 / バイパス %d 件 / 窓 GitHub %d 日・レジストリ %d 日\n\n",
		len(targets), len(skipped), len(bypasses), releaseWindowDays, registryWindowDays)

	if len(policyViolations) > 0 {
		fenced(&b, "バイパス設定の違反", policyViolations)
	}
	if len(blocked) > 0 {
		lines := make([]string, 0, len(blocked))
		for _, f := range blocked {
			lines = append(lines, fmt.Sprintf("- %s（%s）— 公開 %d 日 / 窓 %d 日（%s）",
				f.tool.id(), f.tool.backend, f.ageDays, f.window, f.published.Format(time.DateOnly)))
		}
		fenced(&b, "cooldown 未達", lines)
	}
	if len(reported) > 0 {
		lines := make([]string, 0, len(reported))
		for _, f := range reported {
			lines = append(lines, fmt.Sprintf("- %s（%s）— 公開 %d 日 / 窓 %d 日", f.tool.id(), f.tool.backend, f.ageDays, f.window))
		}
		fenced(&b, "参考: 窓内だがブロックしないもの", lines)
	}
	if len(unresolved) > 0 {
		lines := make([]string, 0, len(unresolved))
		for _, t := range unresolved {
			lines = append(lines, fmt.Sprintf("- %s（%s）", t.id(), t.backend))
		}
		fenced(&b, "公開時刻を取得できなかったもの", lines)
	}
	return b.String()
}

// appendOutput は後続ステップ用に件数を GITHUB_OUTPUT へ書く。
func appendOutput(path string, findings, audited, blocking int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, outputPerm) //nolint:gosec // path は GITHUB_OUTPUT（ランナー提供）
	if err != nil {
		return xerrors.Wrap(err, "open GITHUB_OUTPUT")
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintf(f, "findings=%d\naudited=%d\nblocking=%d\n", findings, audited, blocking)
	return err
}
