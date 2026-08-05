// Package main は go.mod の各モジュールが供給網 cooldown を満たしているかを検査するツール。
//
//	gate:  base からの差分で追加 / 更新された direct モジュールのうち、公開から窓日数未満の
//	       ものを違反として報告し、非ゼロで終了する。
//	audit: go.mod 全件を棚卸しし、窓内のものを報告する。常に 0 で終了する。
//
// **2 分割の理由は npm 側との性格の違いにある。** npm は `.npmrc` の `min-release-age` が
// 依存解決そのものを拒否するため、窓内エントリが lockfile に載ること自体が「防御を外した
// 証拠」になり、CI ツールは検知器で足りる。Go には解決時に窓を強制する機構が無く、`go get` に
// 「新しすぎるから採るな」と言わせる手段が存在しない。したがって gate は検知器ではなく
// **防御の本体**であり、報告に留めれば窓はどこにも存在しないままになる。audit は既存依存の
// 棚卸し用で、gate が grandfather する範囲を可視化するためだけに 0 で終える。
//
// **gate が direct だけを落とすのは MVS の帰結。** indirect の版は direct が要求する下限に
// 縛られ、自分では下げられないことがある。打つ手の無い違反を作らないため indirect は報告に
// 留めるが、攻撃面としては indirect こそ本番なので黙らせはしない。
//
// **窓は 7 日。** Go モジュールには install script が無く `go mod download` は任意コードを
// 実行しないため、npm の「publish 直後に install で即マシン奪取」というクラスは成立しない。
// ここで守るのは「悪意あるコードを気づかず取り込んで出荷する」窓である。値は本リポジトリの
// 実績から採った。過去に `go.mod` を触った 47 コミットのうち窓 7 日で止まるのは 12 件、
// 14 日にしても 15 件で、7 と 14 の間に崖が無い。加えて「cooldown を満たす最新版へ更新する」
// と明示して人間が選んだ実績が 7.4 日であり、既存の運用判断が 7 日相当で回っていた。
//
// **緊急のバイパスは .github/go-cooldown-bypass.toml が受ける。** エントリは期限を必ず持ち、
// 期限切れ・期限過長・対象不在は gate / audit のどちらでも失敗になる。期限は go.mod が
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
	minArgs = 2 // プログラム名 + サブコマンド

	proxyBase  = "https://proxy.golang.org/"
	bypassFile = ".github/go-cooldown-bypass.toml" //nolint:gosec // 資格情報ではなくバイパス lockfile のパス

	defaultWindowDays = 7
	maxBypassMonths   = 3 // バイパスの期限に許す最大の先送り幅

	fetchTimeout = 30 * time.Second
	gitTimeout   = 30 * time.Second
	fetchWorkers = 8
	minFenceLen  = 3 // CommonMark のフェンス下限
	hoursPerDay  = 24
	summaryPerm  = 0o644
	outputPerm   = 0o600
)

var (
	// errProxyStatus は、module proxy が 200 / 404 以外のステータスを返した場合のエラー。
	errProxyStatus = xerrors.New("unexpected module proxy status")
	// errProxyNotFound は、proxy がそのバージョンを知らない場合のエラー。
	errProxyNotFound = xerrors.New("module version not found on the proxy")
	// errBypassInvalidLine は、バイパス lockfile に解釈できない行があった場合のエラー。
	errBypassInvalidLine = xerrors.New("invalid bypass line")
	// errBypassDuplicateKey は、バイパス lockfile にキーの重複があった場合のエラー。
	errBypassDuplicateKey = xerrors.New("duplicate bypass key")

	// requireLineRe は go.mod の require 1 行を module / version / indirect 注釈へ割る。
	requireLineRe = regexp.MustCompile(`^\s*([^\s()]+)\s+(v[^\s]+)(\s+// indirect)?\s*$`)
	// bypassLineRe は `"module@version" = { expires = 2026-11-06, issue = 123, reason = "..." }` を読む。
	// module / version に空白を許さないので、値が改行でログ行やエントリを偽造することはできない。
	bypassLineRe = regexp.MustCompile(
		`^"([^"]+)"\s*=\s*\{\s*expires\s*=\s*(\d{4}-\d{2}-\d{2})\s*,\s*issue\s*=\s*(\d+)\s*,\s*reason\s*=\s*"([^"]*)"\s*\}$`)
)

// requirement は go.mod の require 1 行。
type requirement struct {
	module   string
	version  string
	indirect bool
}

// bypass は cooldown を明示的に外す 1 エントリ。期限・根拠 issue・理由をすべて必須とするのは、
// 誰がいつ何のために外したかを追えないバイパスが、恒久 allowlist と区別できなくなるため。
type bypass struct {
	expires time.Time
	issue   int
	reason  string
	line    int
}

// finding は窓を満たさないモジュール 1 件。
type finding struct {
	req       requirement
	published time.Time
	ageDays   int
}

func (r requirement) key() string { return r.module + "@" + r.version }

func main() {
	log.SetFlags(0)
	if len(os.Args) < minArgs {
		log.Fatalf("❌ usage: go-cooldown <gate|audit> [--base=<git-ref>] [--window-days=N] [--summary-out=<path>] [--github]")
	}
	sub := os.Args[1]
	if sub != "gate" && sub != "audit" {
		log.Fatalf("❌ 未知のサブコマンド %q（gate | audit）", sub) //nolint:gosec // 出力先は CI のログで、値は %q が引用する
	}

	fs := flag.NewFlagSet(sub, flag.ExitOnError)
	base := fs.String("base", "", "gate: この git ref からの差分で追加 / 更新された require だけを対象にする")
	window := fs.Int("window-days", defaultWindowDays, "公開からこの日数に満たないバージョンを窓内とみなす")
	summaryOut := fs.String("summary-out", "", "Markdown サマリの出力先")
	github := fs.Bool("github", false, "GitHub Actions のアノテーションを出力する")
	_ = fs.Parse(os.Args[2:])

	if sub == "gate" && *base == "" {
		log.Fatalf("❌ gate には --base が要ります（比較対象が無いと差分を取れません）")
	}

	ctx := context.Background()
	now := time.Now().UTC()
	// 期限の比較は暦日で行う。時刻を残すと 3 ヶ月の境界が実行時刻とタイムゾーンで動く。
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	current, err := parseGoMod(readWorkTreeGoMod())
	if err != nil {
		log.Fatalf("❌ go.mod: %v", err)
	}

	bypasses, err := readBypasses(bypassFile)
	if err != nil {
		log.Fatalf("❌ %s: %v", bypassFile, err)
	}

	// バイパスの規約違反は gate / audit のどちらでも失敗させる。期限切れは go.mod が
	// 変わらなくても訪れるため、差分の有無で検査を止めると永久に検出されない。
	policyViolations, invalidBypasses := validateBypasses(bypasses, current, today)

	targets := current
	if sub == "gate" {
		targets, err = added(*base, current)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}
	}

	findings, unresolved, err := inspect(ctx, targets, *window, now)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}

	blocking := report(sub, findings, unresolved, policyViolations, bypasses, invalidBypasses, targets, *window, *github)

	if *summaryOut != "" {
		body := summary(sub, findings, unresolved, policyViolations, bypasses, invalidBypasses, targets, *window)
		//nolint:gosec // 出力先はワークフローが渡すパスで、外部入力ではない
		if writeErr := os.WriteFile(*summaryOut, []byte(body), summaryPerm); writeErr != nil {
			log.Fatalf("❌ write summary: %v", writeErr)
		}
	}
	if out := os.Getenv("GITHUB_OUTPUT"); out != "" {
		if writeErr := appendOutput(out, len(findings), len(targets), blocking); writeErr != nil {
			log.Fatalf("❌ write GITHUB_OUTPUT: %v", writeErr)
		}
	}

	if blocking > 0 {
		os.Exit(1)
	}
}

func readWorkTreeGoMod() []byte {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatalf("❌ read go.mod: %v", err)
	}
	return b
}

// parseGoMod は require ブロックと単一行 require の両方から依存を読む。go.mod は direct と
// indirect で require ブロックが分かれるのが通例だが、`go mod tidy` はどちらの形も書き得る。
func parseGoMod(content []byte) ([]requirement, error) {
	var reqs []requirement
	inBlock := false
	sc := bufio.NewScanner(strings.NewReader(string(content)))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if i := strings.Index(trimmed, "//"); i >= 0 && !strings.HasPrefix(strings.TrimSpace(trimmed[i:]), "// indirect") {
			trimmed = strings.TrimSpace(trimmed[:i])
		}
		switch {
		case trimmed == "require (":
			inBlock = true
		case inBlock && trimmed == ")":
			inBlock = false
		case inBlock:
			if m := requireLineRe.FindStringSubmatch(trimmed); m != nil {
				reqs = append(reqs, requirement{module: m[1], version: m[2], indirect: m[3] != ""})
			}
		case strings.HasPrefix(trimmed, "require "):
			if m := requireLineRe.FindStringSubmatch(strings.TrimPrefix(trimmed, "require ")); m != nil {
				reqs = append(reqs, requirement{module: m[1], version: m[2], indirect: m[3] != ""})
			}
		}
	}
	return reqs, sc.Err()
}

// added は base 時点の go.mod に無かった (module, version) の組を返す。バージョンを上げた
// モジュールも新しい組として現れるため、追加と更新をひとつの判定で拾える。
func added(base string, current []requirement) ([]requirement, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "git", "show", base+":go.mod").Output() //nolint:gosec // base は呼び出し側が与える git ref
	if err != nil {
		return nil, xerrors.Wrap(err, fmt.Sprintf("git show %s:go.mod（fetch-depth: 0 で base を取得できていない可能性があります）", base))
	}
	before, err := parseGoMod(out)
	if err != nil {
		return nil, xerrors.Wrap(err, "base の go.mod")
	}
	return diffAdded(before, current), nil
}

// diffAdded は before に無い (module, version) の組を返す。
func diffAdded(before, current []requirement) []requirement {
	known := make(map[string]struct{}, len(before))
	for _, r := range before {
		known[r.key()] = struct{}{}
	}
	var fresh []requirement
	for _, r := range current {
		if _, ok := known[r.key()]; !ok {
			fresh = append(fresh, r)
		}
	}
	return fresh
}

// escapePath は module path を proxy のパス表記へ変換する。proxy は大文字を `!` + 小文字で
// 受けるため、未エスケープのまま問い合わせると 404 になり「存在しないバージョン」に化ける。
func escapePath(path string) string {
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

// publishedAt は module proxy の .info からそのバージョンの公開時刻を取る。GOPROXY プロトコルの
// 一部なので追加の依存を持たずに済む。
func publishedAt(ctx context.Context, client *http.Client, r requirement) (time.Time, error) {
	url := proxyBase + escapePath(r.module) + "/@v/" + r.version + ".info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, xerrors.Wrap(err, "new request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, xerrors.Wrap(err, "fetch "+url)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return time.Time{}, xerrors.Wrap(errProxyNotFound, r.key())
	}
	if resp.StatusCode != http.StatusOK {
		return time.Time{}, xerrors.Wrap(errProxyStatus, fmt.Sprintf("%s: %d", r.key(), resp.StatusCode))
	}

	var info struct {
		Time time.Time `json:"Time"` //nolint:tagliatelle // module proxy の応答フィールド名
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return time.Time{}, xerrors.Wrap(err, "decode "+url)
	}
	return info.Time, nil
}

// inspect は対象モジュールの公開時刻を引き、窓内のものと取得できなかったものを返す。
func inspect(ctx context.Context, targets []requirement, window int, now time.Time) ([]finding, []requirement, error) {
	client := &http.Client{Timeout: fetchTimeout}

	var (
		mu         sync.Mutex
		findings   []finding
		unresolved []requirement
		fatal      error
		wg         sync.WaitGroup
	)
	sem := make(chan struct{}, fetchWorkers)

	for _, r := range targets {
		wg.Add(1)
		go func(r requirement) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			at, err := publishedAt(ctx, client, r)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case xerrors.Is(err, errProxyNotFound):
				unresolved = append(unresolved, r)
			case err != nil:
				if fatal == nil {
					fatal = err
				}
			default:
				if age := int(now.Sub(at).Hours() / hoursPerDay); age < window {
					findings = append(findings, finding{req: r, published: at, ageDays: age})
				}
			}
		}(r)
	}
	wg.Wait()

	if fatal != nil {
		return nil, nil, fatal
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ageDays < findings[j].ageDays })
	sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].key() < unresolved[j].key() })
	return findings, unresolved, nil
}

// readBypasses はバイパス lockfile を module@version→bypass として読む。空行とコメント行以外で
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
				`%d 行目: %q（形式は "<module>@<version>" = { expires = YYYY-MM-DD, issue = <N>, reason = "..." }）`, lineNo, line))
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
// 置くのは、期限を遠い未来へ置くだけで同じ状態を作れてしまうため。
//
// 無効なバイパスは効力も失う。実行はどのみち落ちるので素通ししても穴にはならないが、期限切れの
// エントリが「通します」と報告する状態は、そのモジュールが検査を通ったのか外されただけなのかを
// 読者が区別できなくする。
func validateBypasses(bypasses map[string]bypass, current []requirement, today time.Time) ([]string, map[string]struct{}) {
	inGoMod := make(map[string]struct{}, len(current))
	for _, r := range current {
		inGoMod[r.key()] = struct{}{}
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
			if _, ok := inGoMod[key]; !ok {
				violations = append(violations, fmt.Sprintf(
					"%s:%d %s は go.mod に存在しません。不要になったエントリは消してください",
					bypassFile, b.line, key))
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
// バイパスの有効性は validateBypasses が別途見ているので、ここでは存在するかどうかだけを見る。
func classify(sub string, findings []finding, bypasses map[string]bypass, invalid map[string]struct{}) ([]finding, []finding, []finding) {
	var bypassed, blocking, reported []finding
	for _, f := range findings {
		switch {
		case hasBypass(bypasses, invalid, f.req):
			bypassed = append(bypassed, f)
		case sub == "gate" && !f.req.indirect:
			blocking = append(blocking, f)
		default:
			reported = append(reported, f)
		}
	}
	return bypassed, blocking, reported
}

func hasBypass(bypasses map[string]bypass, invalid map[string]struct{}, r requirement) bool {
	if _, bad := invalid[r.key()]; bad {
		return false
	}
	_, ok := bypasses[r.key()]
	return ok
}

// report は結果を標準出力へ書き、終了コードを非ゼロにすべき件数を返す。audit は窓内の finding では
// 落ちない（既存依存の棚卸しであり、窓内であること自体は誰の違反でもない）。ただしバイパス自身の
// 規約違反だけは audit でも失敗させる。期限切れの回収がスケジュール実行に懸かっているため。
func report(
	sub string, findings []finding, unresolved []requirement,
	policyViolations []string, bypasses map[string]bypass, invalid map[string]struct{},
	targets []requirement, window int, github bool,
) int {
	//nolint:gosec // sub は gate / audit のいずれかに限定済み
	log.Printf("ℹ️ go-cooldown %s: 対象 %d 件 / 窓 %d 日", sub, len(targets), window)

	blockingCount := len(policyViolations)
	for _, v := range policyViolations {
		log.Printf("❌ %s", v)
		if github {
			log.Printf("::error file=%s::%s", bypassFile, v)
		}
	}

	bypassed, blocked, reported := classify(sub, findings, bypasses, invalid)

	for _, f := range bypassed {
		b := bypasses[f.req.key()]
		log.Printf("🔓 %s は公開 %d 日ですが、#%d のバイパスで通します（期限 %s / %s）",
			f.req.key(), f.ageDays, b.issue, b.expires.Format(time.DateOnly), b.reason)
	}

	blockingCount += len(blocked)
	for _, f := range blocked {
		msg := fmt.Sprintf("%s は公開 %d 日で cooldown %d 日を満たしていません。窓明けを待つか、"+
			"待てない根拠を %s へ期限付きで記録してください", f.req.key(), f.ageDays, window, bypassFile)
		//nolint:gosec // module path は空白を許さない正規表現で切り出しており改行を持ち込めない
		log.Printf("❌ %s", msg)
		if github {
			//nolint:gosec // module path は空白を許さない正規表現で切り出しており改行を持ち込めない
			log.Printf("::error file=go.mod::%s", msg)
		}
	}

	for _, f := range reported {
		kind := "direct"
		if f.req.indirect {
			kind = "indirect"
		}
		//nolint:gosec // module path は空白を許さない正規表現で切り出しており改行を持ち込めない
		log.Printf("⚠️ %s（%s）は公開 %d 日で cooldown %d 日を満たしていません", f.req.key(), kind, f.ageDays, window)
		if github {
			//nolint:gosec // module path は空白を許さない正規表現で切り出しており改行を持ち込めない
			log.Printf("::warning file=go.mod::%s（%s）は公開 %d 日で cooldown %d 日を満たしていません",
				f.req.key(), kind, f.ageDays, window)
		}
	}

	for _, r := range unresolved {
		// 取得できないものを黙って通すと、検査が「見た結果 OK」と「見ていない」を区別できなくなる。
		// gate では direct のみ落とす。indirect を落としても打つ手が無いのは窓内の場合と同じ。
		msg := fmt.Sprintf("%s の公開時刻を module proxy から取得できませんでした（private module か、"+
			"proxy に載らない経路の可能性があります）", r.key())
		if sub == "gate" && !r.indirect {
			blockingCount++
			log.Printf("❌ %s", msg)
			if github {
				log.Printf("::error file=go.mod::%s", msg)
			}
			continue
		}
		log.Printf("⚠️ %s", msg)
	}

	if blockingCount == 0 {
		//nolint:gosec // sub は gate / audit のいずれかに限定済み
		log.Printf("✅ go-cooldown %s: 違反なし（バイパス %d 件）", sub, len(bypasses))
	}
	return blockingCount
}

// fenceFor は text を包むのに足りるフェンスを返す。長さは text 中の最長バッククォート連 + 1 で、
// 最低 3。text 側がフェンスを閉じられないことが、この計算の目的である。
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

// fenced は見出しと、フェンスで包んだ本体を書く。module path は go.mod 由来で、この検査が
// 動く時点では pull request が中身を決めている。見出しは読ませたいのでテンプレート側に残し、
// 値の側だけをフェンスへ入れる。
func fenced(b *strings.Builder, heading string, lines []string) {
	body := strings.Join(lines, "\n")
	fence := fenceFor(body)
	fmt.Fprintf(b, "## %s (%d)\n\n%stext\n%s\n%s\n\n", heading, len(lines), fence, body, fence)
}

// summary は GITHUB_STEP_SUMMARY 用の Markdown を組む。
func summary(
	sub string, findings []finding, unresolved []requirement,
	policyViolations []string, bypasses map[string]bypass, invalid map[string]struct{},
	targets []requirement, window int,
) string {
	var b strings.Builder
	_, blocked, reported := classify(sub, findings, bypasses, invalid)

	fmt.Fprintf(&b, "対象 %d 件 / 窓 %d 日 / バイパス %d 件\n\n", len(targets), window, len(bypasses))

	if len(policyViolations) > 0 {
		fenced(&b, "バイパス設定の違反", policyViolations)
	}
	if len(blocked) > 0 {
		lines := make([]string, 0, len(blocked))
		for _, f := range blocked {
			lines = append(lines, fmt.Sprintf("- %s — 公開 %d 日（%s）",
				f.req.key(), f.ageDays, f.published.Format(time.DateOnly)))
		}
		fenced(&b, "cooldown 未達", lines)
	}
	if len(reported) > 0 {
		lines := make([]string, 0, len(reported))
		for _, f := range reported {
			lines = append(lines, fmt.Sprintf("- %s — 公開 %d 日", f.req.key(), f.ageDays))
		}
		fenced(&b, "参考: 窓内だがブロックしないもの", lines)
	}
	if len(unresolved) > 0 {
		lines := make([]string, 0, len(unresolved))
		for _, r := range unresolved {
			lines = append(lines, "- "+r.key())
		}
		fenced(&b, "公開時刻を取得できなかったもの", lines)
	}
	if b.Len() == 0 {
		b.WriteString("違反はありません。\n")
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
