// Package main は GitHub Actions の `uses:` 参照を不変の commit SHA へ固定するツール。
//
//	resolve: .github/workflows/** と .github/actions/** の外部アクション参照を走査し、
//	         tag/version を git ls-remote で SHA へ解決して lockfile (.github/actions-pin.toml) へ書き出す。
//	apply:   lockfile を SSOT に各ファイルの `uses: owner/repo[/sub]@<ref>` を
//	         `uses: owner/repo[/sub]@<sha> # <ref>` へ置換する。
//	check:   apply と同じ判定を書き換えなしで行い、未固定/古い/未登録があれば非ゼロ終了する（CI / hook 用）。
//
// 既に固定済み (`@<sha> # <ref>`) の行は、コメントの <ref> を版として再解決するため idempotent。
// ローカル参照 (`uses: ./...`) は @ を持たないため対象外。
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	lockFile        = ".github/actions-pin.toml"
	filePerm        = 0o644
	minArgs         = 2 // プログラム名 + サブコマンド
	repoSegments    = 2 // owner/repo
	lsRemoteCols    = 2 // <sha>\t<refname>
	lsRemoteTimeout = 30 * time.Second
	hoursPerDay     = 24

	usage = "❌ usage: pin-actions <resolve|apply|check>"
)

var (
	// uses: [-] owner/repo[/sub]@<ref> [# <tag>]
	// 空白は `[ \t]` に限定する（`\s` だと改行を食って行が結合する）。
	usesRe = regexp.MustCompile(`(?m)^([ \t]*(?:-[ \t]*)?uses:[ \t]*)([^@\s]+)@([^\s#]+)(?:[ \t]*#[ \t]*(\S+))?[ \t]*$`)
	lockRe = regexp.MustCompile(`^"([^"]+)"\s*=\s*"([0-9a-f]{40})"`)
	// usesRe の取りこぼしを拾う緩いパターン。行頭にアンカーせず `owner/repo@ref` の形だけを見る。
	looseUsesRe = regexp.MustCompile(`uses:[ \t]*["']?([A-Za-z0-9_.-]+/[^\s"'#,}]+@[^\s"'#,}]+)`)
)

var (
	// errGitHubAPIStatus は、GitHub API が想定外の HTTP ステータスを返した場合のエラー。
	errGitHubAPIStatus = xerrors.New("unexpected GitHub API status")
	// errRefNotFound は、指定 ref が upstream に見つからなかった場合のエラー。
	errRefNotFound = xerrors.New("ref が見つかりません")
	// errRefDateUnavailable は、解決先の経過日数を判定できる日時が一つも得られなかった場合のエラー。
	errRefDateUnavailable = xerrors.New("解決先の日時を取得できません")
	// errLooseUses は、厳密パターンで解釈できない記法の外部アクション参照を検出した場合のエラー。
	errLooseUses = xerrors.New("固定対象として解釈できない記法の uses: があります")
	// errLockInvalidLine は、lockfile に代入として解釈できない行があった場合のエラー。
	errLockInvalidLine = xerrors.New("lockfile に解釈できない行があります")
	// errLockDuplicateKey は、lockfile に同一キーが複数回現れた場合のエラー。
	errLockDuplicateKey = xerrors.New("lockfile にキーの重複があります")
	// errLockOrphanKey は、lockfile にあるがどの uses: からも参照されないエントリがあった場合のエラー。
	errLockOrphanKey = xerrors.New("lockfile に参照されていないエントリがあります")
	// errLockMissingKey は、uses: の参照が lockfile に登録されていない場合のエラー。
	errLockMissingKey = xerrors.New("lockfile に未登録の参照があります")
)

// ref はアクション参照 1 件。repo は owner/repo、sub はサブパス（codeql-action/init 等）、tag は固定対象の版。
type ref struct {
	repo string
	sub  string
	tag  string
}

// rewritePlan は走査結果から導いた固定計画。書き込みは、この計画に問題が無いと確定してから行う。
type rewritePlan struct {
	// changes は書き換えが必要なファイルの絶対パスと固定後の内容。
	changes map[string]string
	// missing は lockfile に未登録だった参照キー。
	missing []string
	// loose は厳密パターンで解釈できなかった uses:（ファイル名付き）。
	loose []string
	// used は走査対象から実際に参照された lockfile のキー。
	used map[string]bool
}

func (r ref) key() string { return r.repo + "@" + r.tag }

func main() {
	log.SetFlags(0)
	if len(os.Args) < minArgs {
		log.Fatal(usage)
	}
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ getwd: %v", err)
	}
	files, err := targetFiles(root)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	switch os.Args[1] {
	case "resolve":
		fs := flag.NewFlagSet("resolve", flag.ExitOnError)
		minAge := fs.Int("min-age-days", 0, "N 日未満のコミットは quarantine（0 で無効）")
		_ = fs.Parse(os.Args[2:])
		resolve(root, files, *minAge)
	case "apply":
		applyOrCheck(root, files, false)
	case "check":
		applyOrCheck(root, files, true)
	default:
		log.Fatal(usage)
	}
}

// parseUses は uses: 行の path / ref / コメント tag から ref を構築する。第2戻り値が false なら対象外。
func parseUses(path, refStr, comment string) (ref, bool) {
	if strings.HasPrefix(path, ".") {
		return ref{}, false // ローカル参照
	}
	seg := strings.Split(path, "/")
	if len(seg) < repoSegments {
		return ref{}, false
	}
	tag := refStr
	if comment != "" {
		tag = comment // 既に SHA 固定済み。版はコメント側。
	}
	return ref{
		repo: seg[0] + "/" + seg[1],
		sub:  strings.Join(seg[repoSegments:], "/"),
		tag:  tag,
	}, true
}

// targetFiles は走査対象の workflow / composite action ファイルを返す。
// composite action は `uses: ./.github/actions/<group>/<name>` のように入れ子へ置けるため再帰的に集める。
// 1 階層の glob では入れ子のファイルが警告なく走査から消え、そこに書かれた外部参照が
// 検疫・固定・drift 検査のいずれの対象からも外れる。
func targetFiles(root string) ([]string, error) {
	var files []string
	for _, pat := range []string{
		".github/workflows/*.yml", ".github/workflows/*.yaml",
	} {
		m, err := filepath.Glob(filepath.Join(root, pat))
		if err != nil {
			return nil, xerrors.Wrap(err, "glob "+pat)
		}
		files = append(files, m...)
	}

	actionsDir := filepath.Join(root, ".github", "actions")
	err := filepath.WalkDir(actionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (d.Name() == "action.yml" || d.Name() == "action.yaml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !xerrors.Is(err, os.ErrNotExist) {
		return nil, xerrors.Wrap(err, "walk .github/actions")
	}

	sort.Strings(files)
	return files, nil
}

// fileRefs は data 中の uses: から固定対象の参照を返す（同一キーの重複を含む）。
func fileRefs(data string) []ref {
	var refs []ref
	for _, m := range usesRe.FindAllStringSubmatch(data, -1) {
		if r, ok := parseUses(m[2], m[3], m[4]); ok {
			refs = append(refs, r)
		}
	}
	return refs
}

// detectLooseUses は usesRe が取りこぼす記法で書かれた外部アクション参照を返す。
//
// usesRe は行頭にアンカーしたブロック記法しか一致しないが、steps は
// `- {name: Checkout, uses: actions/checkout@v4}` のような YAML flow mapping でも正当に書ける。
// この記法は一致ゼロになり「固定漏れ無し」と区別が付かないため、厳密パターンで解釈できた uses: を
// 取り除いたうえで残りを緩いパターンで拾い、外部参照が残っていれば呼び出し元が fail-close する。
// ローカル参照と版を持たない参照は誤検知を避けるため除外する。
func detectLooseUses(data string) []string {
	var found []string
	for line := range strings.SplitSeq(usesRe.ReplaceAllString(data, ""), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, m := range looseUsesRe.FindAllStringSubmatch(line, -1) {
			if strings.HasPrefix(m[1], ".") {
				continue
			}
			found = append(found, m[1])
		}
	}
	sort.Strings(found)
	return uniq(found)
}

func resolve(root string, files []string, minAgeDays int) {
	keys, err := collectKeys(root, files)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	// lockfile 不在（初回）は空マップで続行するが、それ以外の読み込み失敗は握り潰さず fail-close する
	// （既存ピンが lock から脱落し供給網ガードの維持保証が破れるのを防ぐ。applyOrCheck と対称）。
	existing, err := readLock(filepath.Join(root, lockFile))
	if !isIgnorableLockErr(err) {
		log.Fatalf("❌ read lockfile: %v", err)
	}

	ctx := context.Background()
	lock := map[string]string{}
	var notes []string
	for k, r := range keys {
		sha, err := resolveSHA(ctx, r.repo, r.tag)
		if err != nil {
			log.Fatalf("❌ resolve %s: %v", k, err)
		}
		ageFn := func() (int, error) { return refAgeDays(ctx, r.repo, r.tag, sha) }
		use, note, err := quarantine(ageFn, k, sha, minAgeDays, existing)
		if err != nil {
			log.Fatalf("❌ age %s: %v", k, err)
		}
		if note != "" {
			notes = append(notes, note)
		}
		if use == "" {
			continue
		}
		lock[k] = use
		log.Printf("  %s -> %s", k, use)
	}
	sort.Strings(notes)
	for _, n := range notes {
		log.Printf("  ⚠️ %s", n)
	}

	if err := writeLock(filepath.Join(root, lockFile), lock); err != nil {
		log.Fatalf("❌ write lockfile: %v", err)
	}
	log.Printf("✅ %s に %d 件を書き出しました", lockFile, len(lock))
}

func collectKeys(root string, files []string) (map[string]ref, error) {
	keys := map[string]ref{}
	var loose []string
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			return nil, xerrors.Wrap(err, "read "+rel(root, f))
		}
		for _, r := range fileRefs(string(data)) {
			keys[r.key()] = r
		}
		for _, r := range detectLooseUses(string(data)) {
			loose = append(loose, rel(root, f)+": "+r)
		}
	}
	if len(loose) > 0 {
		return nil, xerrors.Wrap(errLooseUses, strings.Join(loose, ", "))
	}
	return keys, nil
}

// quarantine は minAgeDays 未満の新しすぎる解決先を採用しない。第1戻り値（採用 SHA）が "" は skip（初回かつ新しすぎ）。
// 経過日数は ageFn から取得し、その失敗は err で呼び出し元へ伝播する。
// minAgeDays<=0 のときは ageFn を呼ばず候補をそのまま採用する。
func quarantine(ageFn func() (int, error), key, candidate string, minAgeDays int, existing map[string]string) (string, string, error) {
	if minAgeDays <= 0 {
		return candidate, "", nil
	}
	age, err := ageFn()
	if err != nil {
		return "", "", err
	}
	if age >= minAgeDays {
		return candidate, "", nil
	}
	if prev, ok := existing[key]; ok {
		return prev, fmt.Sprintf("%s: 解決先が %d 日 (<%d) のため既存ピンを維持", key, age, minAgeDays), nil
	}
	return "", fmt.Sprintf("%s: 解決先が %d 日 (<%d)・既存ピン無しのため skip", key, age, minAgeDays), nil
}

// refAgeDays は解決先の経過日数を返す。tag に対応する Release の published_at と sha の commit 日時を
// 常に両方取得し、新しい方から算出する。
func refAgeDays(ctx context.Context, repo, tag, sha string) (int, error) {
	var release struct {
		//nolint:tagliatelle // GitHub API のレスポンスフィールド名(published_at)に合わせる必要があるため
		PublishedAt time.Time `json:"published_at"`
	}
	st, err := githubGet(ctx, "https://api.github.com/repos/"+repo+"/releases/tags/"+url.PathEscape(tag), &release)
	if err != nil {
		return 0, err
	}
	if st != http.StatusOK && st != http.StatusNotFound {
		return 0, xerrors.Wrap(errGitHubAPIStatus, fmt.Sprintf("releases/tags/%s status=%d", tag, st))
	}
	var published time.Time
	if st == http.StatusOK {
		published = release.PublishedAt
	}

	var commit struct {
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	st, err = githubGet(ctx, "https://api.github.com/repos/"+repo+"/commits/"+url.PathEscape(sha), &commit)
	if err != nil {
		return 0, err
	}
	if st != http.StatusOK {
		return 0, xerrors.Wrap(errGitHubAPIStatus, fmt.Sprintf("commits/%s status=%d", sha, st))
	}

	t, err := pickRefTime(published, commit.Commit.Committer.Date)
	if err != nil {
		return 0, err
	}
	return daysSince(t), nil
}

// pickRefTime は Release の公開日時と commit の日時のうち新しい方を返す。どちらも不明ならエラー。
//
// Release は tag 名にしか紐づかず、tag が別の commit へ付け替えられても published_at は据え置かれる。
// published_at だけを見ると、この機構が想定する脅威そのもの（侵害されたアカウントによる moving tag の
// 付け替え）に対して何年も前の日付が返り、検疫が常に素通りする。逆に commit の committer date は
// GIT_COMMITTER_DATE で任意の過去日時へ設定できる。どちらか一方でも「新しい」と言う限り検疫が掛かる
// よう新しい方を採る。日付偽装そのものに耐える保証ではなく、付け替えの検知は lockfile の差分レビューが
// 担う（docs/design/security.md）。
func pickRefTime(published, committed time.Time) (time.Time, error) {
	switch {
	case published.IsZero() && committed.IsZero():
		return time.Time{}, errRefDateUnavailable
	case published.After(committed):
		return published, nil
	default:
		return committed, nil
	}
}

// githubGet は GitHub API を GET し、200 のとき out に JSON をデコードして HTTP ステータスを返す。
func githubGet(ctx context.Context, url string, out any) (int, error) {
	cctx, cancel := context.WithTimeout(ctx, lsRemoteTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func daysSince(t time.Time) int {
	return int(time.Since(t).Hours() / hoursPerDay)
}

// rewritePins は lock を元に uses: を固定した内容と、lock 未登録の参照キー一覧を返す。
func rewritePins(data string, lock map[string]string) (string, []string) {
	var missing []string
	out := usesRe.ReplaceAllStringFunc(data, func(line string) string {
		m := usesRe.FindStringSubmatch(line)
		r, ok := parseUses(m[2], m[3], m[4])
		if !ok {
			return line
		}
		sha, found := lock[r.key()]
		if !found {
			missing = append(missing, r.key())
			return line
		}
		path := r.repo
		if r.sub != "" {
			path += "/" + r.sub
		}
		return fmt.Sprintf("%s%s@%s # %s", m[1], path, sha, r.tag)
	})
	return out, missing
}

// planRewrites は全ファイルを読み切り、固定後の内容と fail-close 条件を確定させる。
// 1 ファイルずつ書きながら進むと、未登録参照で中断したときに「exit 1 なのに作業ツリーは書き換え済み」
// という中途半端な状態が残るため、判定と書き込みを分ける。
func planRewrites(root string, files []string, lock map[string]string) (*rewritePlan, error) {
	plan := &rewritePlan{changes: map[string]string{}, used: map[string]bool{}}
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			return nil, xerrors.Wrap(err, "read "+rel(root, f))
		}
		for _, r := range detectLooseUses(string(data)) {
			plan.loose = append(plan.loose, rel(root, f)+": "+r)
		}
		for _, r := range fileRefs(string(data)) {
			plan.used[r.key()] = true
		}
		out, miss := rewritePins(string(data), lock)
		plan.missing = append(plan.missing, miss...)
		if out != string(data) {
			plan.changes[f] = out
		}
	}
	return plan, nil
}

// validate は書き込み前に中断すべき条件を返す。
func (p *rewritePlan) validate(lock map[string]string) error {
	if len(p.loose) > 0 {
		sort.Strings(p.loose)
		return xerrors.Wrap(errLooseUses, strings.Join(uniq(p.loose), ", "))
	}
	if len(p.missing) > 0 {
		sort.Strings(p.missing)
		return xerrors.Wrap(errLockMissingKey,
			strings.Join(uniq(p.missing), ", ")+"（make pin-actions-resolve を実行してください）")
	}
	if orphans := orphanKeys(lock, p.used); len(orphans) > 0 {
		return xerrors.Wrap(errLockOrphanKey,
			strings.Join(orphans, ", ")+"（make pin-actions-resolve を実行するか該当行を削除してください）")
	}
	return nil
}

// orphanKeys は lockfile にあるがどの uses: からも参照されないキーを返す。
// 孤児を残すと lockfile が現用インベントリの鏡でなくなり、レビューで lock の差分だけを読めば足りるという
// 前提が崩れる。
func orphanKeys(lock map[string]string, used map[string]bool) []string {
	var orphans []string
	for k := range lock {
		if !used[k] {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// applyOrCheck は lockfile を SSOT に uses: を固定する。dryRun=true は書き換えず drift を非ゼロ終了で報告する。
func applyOrCheck(root string, files []string, dryRun bool) {
	lock, err := readLock(filepath.Join(root, lockFile))
	if err != nil {
		log.Fatalf("❌ read lockfile（先に resolve を実行してください）: %v", err)
	}
	plan, err := planRewrites(root, files, lock)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	if err := plan.validate(lock); err != nil {
		log.Fatalf("❌ %v", err)
	}

	paths := make([]string, 0, len(plan.changes))
	for f := range plan.changes {
		paths = append(paths, f)
	}
	sort.Strings(paths)

	if dryRun {
		if len(paths) > 0 {
			drifted := make([]string, 0, len(paths))
			for _, f := range paths {
				drifted = append(drifted, rel(root, f))
			}
			log.Fatalf("❌ 未固定/古い参照があります（make pin-actions-resolve && make pin-actions-apply してコミットしてください）: %s",
				strings.Join(drifted, ", "))
		}
		log.Printf("✅ 全アクションが lockfile 通りに固定されています")
		return
	}
	for _, f := range paths {
		if err := os.WriteFile(f, []byte(plan.changes[f]), filePerm); err != nil { //nolint:gosec // workflow ファイル権限
			log.Fatalf("❌ write %s: %v", rel(root, f), err)
		}
		log.Printf("  updated %s", rel(root, f))
	}
	log.Printf("✅ %d ファイルを固定しました", len(paths))
}

// resolveSHA は owner/repo の tag/branch を commit SHA へ解決する。annotated tag は ^{} で deref する。
func resolveSHA(ctx context.Context, repo, tag string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, lsRemoteTimeout)
	defer cancel()
	url := "https://github.com/" + repo
	// --end-of-options 以降を確実に ref 名として扱わせ、`-` 始まりの tag のオプション誤解釈を防ぐ。
	out, err := exec.CommandContext(cctx, "git", "ls-remote", url, "--end-of-options", tag, tag+"^{}").Output() //nolint:gosec // 参照名は workflow 由来
	if err != nil {
		return "", xerrors.Wrap(err, "git ls-remote")
	}
	return selectSHA(string(out), tag)
}

// selectSHA は git ls-remote の生出力から tag に対応する commit SHA を選ぶ。
// 優先順位は annotated tag の deref (^{}) > 軽量 tag > branch head。いずれも無ければ未発見エラー。
func selectSHA(out, tag string) (string, error) {
	var tagSHA, derefSHA, headSHA string
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) != lsRemoteCols {
			continue
		}
		sha, name := parts[0], parts[1]
		switch name {
		case "refs/tags/" + tag + "^{}":
			derefSHA = sha
		case "refs/tags/" + tag:
			tagSHA = sha
		case "refs/heads/" + tag:
			headSHA = sha
		}
	}
	switch {
	case derefSHA != "": // annotated tag は deref した commit を採用
		return derefSHA, nil
	case tagSHA != "":
		return tagSHA, nil
	case headSHA != "":
		return headSHA, nil
	default:
		return "", xerrors.Wrap(errRefNotFound, fmt.Sprintf("%q", tag))
	}
}

func writeLock(path string, lock map[string]string) error {
	keys := make([]string, 0, len(lock))
	for k := range lock {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# GitHub Actions の pin 対象 SHA（SSOT）。\n")
	b.WriteString("# make pin-actions-resolve で解決・make pin-actions-apply で workflow へ反映する。\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%q = %q\n", k, lock[k])
	}
	return os.WriteFile(path, []byte(b.String()), filePerm)
}

// isIgnorableLockErr は、lockfile 読み込みエラーのうち無視して続行してよいものを判定する。
// nil（成功）と「ファイル不在」（初回 resolve）のみ true。それ以外（権限エラー・読み取り失敗等）は fail-close 対象。
func isIgnorableLockErr(err error) bool {
	return err == nil || xerrors.Is(err, os.ErrNotExist)
}

// readLock は lockfile を repo@tag→SHA として読む。空行とコメント行以外で代入として解釈できない行、
// および既出キーの再定義はエラーにする。読み飛ばしや後勝ちの上書きは、そのエントリが「存在しない」
// あるいは「行順で決まる」状態を警告なく作り、lockfile が SSOT として機能しなくなる。
func readLock(path string) (map[string]string, error) {
	f, err := os.Open(path) //nolint:gosec // path is constructed from cwd + literal filename
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	lock := map[string]string{}
	sc := bufio.NewScanner(f)
	for lineNo := 1; sc.Scan(); lineNo++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := lockRe.FindStringSubmatch(line)
		if m == nil {
			return nil, xerrors.Wrap(errLockInvalidLine,
				fmt.Sprintf("%d 行目: %q（make pin-actions-resolve を実行するか該当行を削除してください）", lineNo, line))
		}
		if _, dup := lock[m[1]]; dup {
			return nil, xerrors.Wrap(errLockDuplicateKey,
				fmt.Sprintf("%d 行目: %q（make pin-actions-resolve を実行するか重複行を削除してください）", lineNo, m[1]))
		}
		lock[m[1]] = m[2]
	}
	return lock, sc.Err()
}

func rel(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

func uniq(s []string) []string {
	out := s[:0]
	var prev string
	for i, v := range s {
		if i == 0 || v != prev {
			out = append(out, v)
		}
		prev = v
	}
	return out
}
