// Package main は package-lock.json のエントリが、その lockfile 自身の
// .npmrc `min-release-age` を満たしているかを監査するツール。
//
//	audit: 各 package-lock.json を走査し、公開から min-release-age 日未満の
//	       バージョンが記録されていないかを調べる。
//
// npm の `min-release-age` は依存解決（npm install / update）時にだけ効き、
// lockfile を再現するだけの `npm ci` では評価されない。本リポジトリの CI と
// イメージビルドは全て `npm ci` なので、cooldown を破って作られた lockfile は
// CI 上では何の異常も示さずに通る。このツールはその死角を埋める。
//
// 検出＝バイパスの証拠になる。cooldown 下の `npm install` は窓内バージョンの
// 解決自体を拒否するため、窓内エントリが lockfile に載る経路は
// 「cooldown を明示的に外して解決した」以外に存在しない。誤検知はほぼ出ない。
//
// **本ツールは finding があっても正常終了する。** cooldown のバイパスは
// テックリード / アーキテクトの専任判断であり、CRITICAL への即応など正当な
// 行使を CI が塞ぐのは誤り。非ゼロ終了はツール自身の失敗（取得不能・解析不能）
// に限る。ゲートではなく検知器であることを、ワークフロー側の設定ではなく
// ツールの性質として持たせている。
//
// 守備範囲は「方針のドリフト」に限る。すなわち事故によるバイパス、規約の風化、
// npm 側の挙動変化である。commit 権限を持つ攻撃者はこのツールもワークフローも
// 同じ変更で削除できるため、技術的な防止手段ではない。成立するのは検知と
// attribution までで、そこから先の抑止は組織側の運用に委ねる。
// 強制の担保は CODEOWNERS による lockfile / .npmrc のレビュー必須化が担う。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	registryBase   = "https://registry.npmjs.org/"
	fetchTimeout   = 30 * time.Second
	gitTimeout     = 30 * time.Second
	fetchWorkers   = 8
	hoursPerDay    = 24
	nodeModulesSep = "node_modules/"
	summaryPerm    = 0o644
	outputPerm     = 0o600
)

var (
	// errRegistryStatus は、npm registry が 200 / 404 以外のステータスを返した場合のエラー。
	errRegistryStatus = xerrors.New("unexpected npm registry status")
	// errUsage は、サブコマンドやフラグの与え方が誤っていることを表すエラー。
	errUsage = xerrors.New("invalid usage")
)

// entry は lockfile に記録された 1 パッケージ。path は lockfile 内のキー（差分表示用に保持する）。
type entry struct {
	path    string
	name    string
	version string
}

// finding は cooldown を満たさないエントリ 1 件。
type finding struct {
	lockfile  string
	entry     entry
	published time.Time
	ageDays   int
	minAge    int
}

func (e entry) key() string { return e.name + "@" + e.version }

// main はエラーを終了コードへ変換するだけに留め、判断は run が持ちます。
// main は 1:1 の対象外でテストを書けないため、ここに分岐を置くと検査されない
// コードがそのぶん増える。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:], &http.Client{Timeout: fetchTimeout}, os.Getwd); err != nil {
		log.Fatalf("%v", err)
	}
}

// run は lockfile を監査して結果を報告します。client は registry への問い合わせ手段、
// wd は走査の起点となるディレクトリの取得手段で、差し替えられるよう引数で受けます。
//
// finding があっても nil を返します。返すエラーはツール自身の失敗（取得不能・書き出し不能）に
// 限られ、cooldown 違反の検出は終了コードへ影響しません（パッケージ doc の設計意図）。
func run(args []string, client *http.Client, wd func() (string, error)) error {
	if len(args) == 0 || args[0] != "audit" {
		return xerrors.Wrap(errUsage, "❌ usage: npm-cooldown audit [--base=<git-ref>] [--summary-out=<path>] [--github]")
	}

	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	base := fs.String("base", "", "この git ref からの差分で追加/変更されたエントリだけを対象にする（未指定なら全件）")
	summaryOut := fs.String("summary-out", "", "Markdown サマリの出力先")
	github := fs.Bool("github", false, "GitHub Actions の ::warning:: アノテーションを出力する")
	if err := fs.Parse(args[1:]); err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return xerrors.Wrap(err, "❌ フラグを解釈できません")
	}

	root, err := wd()
	if err != nil {
		return xerrors.Wrap(err, "❌ getwd")
	}

	locks, err := lockfiles(root)
	if err != nil {
		return xerrors.Wrap(err, "❌ lockfile の走査")
	}
	if len(locks) == 0 {
		log.Printf("ℹ️ package-lock.json が見つかりません")

		return nil
	}

	findings, audited, err := auditAll(context.Background(), client, root, locks, *base)
	if err != nil {
		return xerrors.Wrap(err, "❌ 監査")
	}

	report(findings, audited, *base, *github)
	if *summaryOut != "" {
		if writeErr := os.WriteFile(*summaryOut, []byte(summary(findings, audited, *base)), summaryPerm); writeErr != nil {
			return xerrors.Wrap(writeErr, "❌ write summary")
		}
	}
	if out := os.Getenv("GITHUB_OUTPUT"); out != "" {
		if writeErr := appendOutput(out, len(findings), audited); writeErr != nil {
			return xerrors.Wrap(writeErr, "❌ write GITHUB_OUTPUT")
		}
	}

	// finding があっても nil を返す（パッケージ doc の設計意図）。
	return nil
}

// lockfiles は node_modules / vendor 配下を除く package-lock.json を返す。
func lockfiles(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, perr := lockPath(root, path, d)
		if perr != nil {
			return perr
		}
		if rel != "" {
			found = append(found, rel)
		}
		return nil
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "walk")
	}
	sort.Strings(found)
	return found, nil
}

// lockPath は走査中の 1 エントリを、収集すべき root からの相対パスへ変換する。対象外なら空文字を、
// 枝ごと落とすディレクトリでは filepath.SkipDir を返す。相対化に失敗したものを絶対パスのまま
// 収集すると、以降の root との結合が壊れた場所を指す。
func lockPath(root, path string, d os.DirEntry) (string, error) {
	if d.IsDir() {
		switch d.Name() {
		case "node_modules", "vendor", ".git":
			return "", filepath.SkipDir
		}
		return "", nil
	}
	if d.Name() != "package-lock.json" {
		return "", nil
	}
	return filepath.Rel(root, path)
}

// minReleaseAge は lockfile と同じディレクトリの .npmrc から min-release-age を読む。
// 宣言が無ければ 0 を返す（＝そのlockfileには守るべき cooldown が無い）。
func minReleaseAge(root, lock string) (int, error) {
	p := filepath.Join(root, filepath.Dir(lock), ".npmrc")
	b, err := os.ReadFile(p) //nolint:gosec // path は lockfile の同階層に限定
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, xerrors.Wrap(err, "read "+p)
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != "min-release-age" {
			continue
		}
		n, cerr := strconv.Atoi(strings.TrimSpace(v))
		if cerr != nil {
			return 0, xerrors.Wrap(cerr, "parse min-release-age in "+p)
		}
		return n, nil
	}
	return 0, nil
}

// parseLock は lockfile(v2/v3) の packages から実パッケージのエントリを取り出す。
// ルート（キーが空文字）と workspace リンクは対象外。
func parseLock(data []byte) ([]entry, error) {
	var doc struct {
		Packages map[string]struct {
			Version string `json:"version"`
			Link    bool   `json:"link"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, xerrors.Wrap(err, "unmarshal lockfile")
	}
	var out []entry
	for path, p := range doc.Packages {
		if path == "" || p.Link || p.Version == "" {
			continue
		}
		idx := strings.LastIndex(path, nodeModulesSep)
		if idx < 0 {
			continue // node_modules 配下でない＝workspace 自身
		}
		out = append(out, entry{path: path, name: path[idx+len(nodeModulesSep):], version: p.Version})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

// changedOnly は base ref 時点の lockfile と比較し、追加/変更された name@version だけを残す。
// base に lockfile が存在しない（新規追加）場合は全件を対象にする。
func changedOnly(ctx context.Context, root, lock, base string, cur []entry) ([]entry, error) {
	cctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "git", "show", base+":"+lock) //nolint:gosec // base は CI 由来の ref、lock は走査結果
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// base 側に lockfile が無い＝新規追加。全件が「追加分」なのでそのまま返す。
		return cur, nil
	}
	prev, err := parseLock(out)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(prev))
	for _, e := range prev {
		known[e.key()] = struct{}{}
	}
	var diff []entry
	for _, e := range cur {
		if _, ok := known[e.key()]; !ok {
			diff = append(diff, e)
		}
	}
	return diff, nil
}

// auditAll は全 lockfile を監査し、finding と監査したエントリ総数を返す。
func auditAll(ctx context.Context, client *http.Client, root string, locks []string, base string) ([]finding, int, error) {
	var findings []finding
	audited := 0

	for _, lock := range locks {
		fs, n, err := auditLock(ctx, client, root, lock, base)
		if err != nil {
			return nil, 0, err
		}
		findings = append(findings, fs...)
		audited += n
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].ageDays < findings[j].ageDays })
	return findings, audited, nil
}

// auditLock は lockfile 1 本を監査する。min-release-age の宣言が無ければ対象外として何も返さない。
func auditLock(ctx context.Context, client *http.Client, root, lock, base string) ([]finding, int, error) {
	minAge, err := minReleaseAge(root, lock)
	if err != nil {
		return nil, 0, err
	}
	if minAge <= 0 {
		log.Printf("⏭️  %s: .npmrc に min-release-age が無いため対象外", lock)
		return nil, 0, nil
	}

	entries, err := lockEntries(ctx, root, lock, base)
	if err != nil {
		return nil, 0, err
	}
	if len(entries) == 0 {
		log.Printf("✅ %s: 対象エントリなし", lock)
		return nil, 0, nil
	}

	times, err := publishTimes(ctx, client, entries)
	if err != nil {
		return nil, 0, err
	}

	var findings []finding
	for _, e := range entries {
		pub, ok := times[e.key()]
		if !ok || pub.IsZero() {
			continue // registry に公開日が無い（private / 取得不能）。判定不能は報告しない
		}
		if age := int(time.Since(pub).Hours() / hoursPerDay); age < minAge {
			findings = append(findings, finding{lockfile: lock, entry: e, published: pub, ageDays: age, minAge: minAge})
		}
	}
	log.Printf("🔍 %s: %d 件を監査（min-release-age=%d）", lock, len(entries), minAge)
	return findings, len(entries), nil
}

// lockEntries は監査対象のエントリを返す。base 指定時はそこからの追加/変更分に絞る。
func lockEntries(ctx context.Context, root, lock, base string) ([]entry, error) {
	data, err := os.ReadFile(filepath.Join(root, lock)) //nolint:gosec // lockfiles() の走査結果
	if err != nil {
		return nil, xerrors.Wrap(err, "read "+lock)
	}
	entries, err := parseLock(data)
	if err != nil {
		return nil, err
	}
	if base == "" {
		return entries, nil
	}
	return changedOnly(ctx, root, lock, base, entries)
}

// publishTimes は name@version の公開日時を registry から並列取得する。
// packument の .time にのみ公開日があるため version 文書では代替できない。
func publishTimes(ctx context.Context, client *http.Client, entries []entry) (map[string]time.Time, error) {
	byName := map[string][]entry{}
	for _, e := range entries {
		byName[e.name] = append(byName[e.name], e)
	}

	var (
		mu      sync.Mutex
		result  = map[string]time.Time{}
		firstEr error
		wg      sync.WaitGroup
		sem     = make(chan struct{}, fetchWorkers)
	)
	for name, es := range byName {
		wg.Add(1)
		go func(name string, es []entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			times, err := packumentTimes(ctx, client, name)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstEr == nil {
					firstEr = err
				}
				return
			}
			for _, e := range es {
				if t, ok := times[e.version]; ok {
					result[e.key()] = t
				}
			}
		}(name, es)
	}
	wg.Wait()
	if firstEr != nil {
		return nil, firstEr
	}
	return result, nil
}

// packumentTimes は 1 パッケージの全バージョンの公開日時を返す。
func packumentTimes(ctx context.Context, client *http.Client, name string) (map[string]time.Time, error) {
	cctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, registryBase+name, nil)
	if err != nil {
		return nil, xerrors.Wrap(err, "request "+name)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, xerrors.Wrap(err, "fetch "+name)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return map[string]time.Time{}, nil // private / 削除済み。判定不能として扱う
	}
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Wrap(errRegistryStatus, fmt.Sprintf("registry %s status=%d", name, resp.StatusCode))
	}

	var doc struct {
		Time map[string]time.Time `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, xerrors.Wrap(err, "decode packument "+name)
	}
	return doc.Time, nil
}

// report は人間向け出力と、GitHub Actions のアノテーションを出す。
func report(findings []finding, audited int, base string, github bool) {
	scope := "全件"
	if base != "" {
		scope = base + " からの追加/変更分"
	}
	if len(findings) == 0 {
		log.Printf("✅ cooldown 違反なし（%s / %d 件を監査）", scope, audited)
		return
	}

	log.Printf("⚠️ cooldown 違反 %d 件（%s / %d 件を監査）", len(findings), scope, audited)
	for _, f := range findings {
		log.Printf("  - %s %s: 公開 %d 日 (< %d) [%s]",
			f.entry.name, f.entry.version, f.ageDays, f.minAge, f.lockfile)
		if github {
			log.Printf("::warning file=%s::%s@%s は公開 %d 日で min-release-age=%d を満たしていません。npm の cooldown を外して解決した lockfile の可能性があります",
				f.lockfile, f.entry.name, f.entry.version, f.ageDays, f.minAge)
		}
	}
}

// summary は PR コメント用の Markdown を組み立てる。
func summary(findings []finding, audited int, base string) string {
	var b strings.Builder
	scope := "全エントリ"
	if base != "" {
		scope = "この変更で追加/変更されたエントリ"
	}
	if len(findings) == 0 {
		fmt.Fprintf(&b, "%s %d 件を監査し、cooldown 違反はありませんでした。\n", scope, audited)
		return b.String()
	}

	fmt.Fprintf(&b, "%s %d 件のうち %d 件が min-release-age を満たしていません。\n\n", scope, audited, len(findings))
	for _, f := range findings {
		fmt.Fprintf(&b, "- `%s@%s` — 公開 %d 日 (< %d) / %s / %s\n",
			f.entry.name, f.entry.version, f.ageDays, f.minAge, f.published.Format(time.RFC3339), f.lockfile)
	}
	b.WriteString("\n")
	b.WriteString("npm の cooldown は `npm install` の依存解決時にしか効かず、CI とイメージビルドが使う `npm ci` では評価されません。\n")
	b.WriteString("そのため窓内バージョンが lockfile に載る経路は「cooldown を明示的に外して解決した」以外にありません。\n")
	b.WriteString("意図的なバイパス（CRITICAL への即応など）であれば、その判断根拠を PR に残してください。心当たりが無い場合は lockfile の出所を確認してください。\n")
	return b.String()
}

// appendOutput は後続ステップ用に findings / audited を GITHUB_OUTPUT へ書く。
func appendOutput(path string, findings, audited int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, outputPerm) //nolint:gosec // path は GITHUB_OUTPUT（ランナー提供）
	if err != nil {
		return xerrors.Wrap(err, "open GITHUB_OUTPUT")
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintf(f, "findings=%d\naudited=%d\n", findings, audited)
	return err
}
