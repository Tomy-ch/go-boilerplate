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
	"log"
	"net/http"
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
)

// ref はアクション参照 1 件。repo は owner/repo、sub はサブパス（codeql-action/init 等）、tag は固定対象の版。
type ref struct {
	repo string
	sub  string
	tag  string
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

func targetFiles(root string) ([]string, error) {
	var files []string
	for _, pat := range []string{
		".github/workflows/*.yml", ".github/workflows/*.yaml",
		".github/actions/*/action.yml", ".github/actions/*/action.yaml",
	} {
		m, err := filepath.Glob(filepath.Join(root, pat))
		if err != nil {
			return nil, xerrors.Wrap(err, "glob "+pat)
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	return files, nil
}

func resolve(root string, files []string, minAgeDays int) {
	keys := collectKeys(files)
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

func collectKeys(files []string) map[string]ref {
	keys := map[string]ref{}
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			log.Fatalf("❌ read %s: %v", f, err)
		}
		for _, m := range usesRe.FindAllStringSubmatch(string(data), -1) {
			if r, ok := parseUses(m[2], m[3], m[4]); ok {
				keys[r.key()] = r
			}
		}
	}
	return keys
}

// quarantine は minAgeDays 未満の新しすぎる解決先を採用しない。第1戻り値（採用 SHA）が "" は skip（初回かつ新しすぎ）。
// 経過日数は ageFn 経由で取得し（I/O を注入して分岐をテスト可能にする）、その失敗は err で呼び出し元へ伝播する。
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

// refAgeDays は解決先の経過日数を返す。タグに対応する Release があれば published_at（偽装困難）を、
// 無ければ commit の committer date をフォールバックに使う。
func refAgeDays(ctx context.Context, repo, tag, sha string) (int, error) {
	var rel struct {
		//nolint:tagliatelle // GitHub API のレスポンスフィールド名(published_at)に合わせる必要があるため
		PublishedAt time.Time `json:"published_at"`
	}
	st, err := githubGet(ctx, "https://api.github.com/repos/"+repo+"/releases/tags/"+tag, &rel)
	if err != nil {
		return 0, err
	}
	if st == http.StatusOK && !rel.PublishedAt.IsZero() {
		return daysSince(rel.PublishedAt), nil
	}
	if st != http.StatusOK && st != http.StatusNotFound {
		return 0, xerrors.New(fmt.Sprintf("releases/tags/%s: %d", tag, st))
	}

	var commit struct {
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	st, err = githubGet(ctx, "https://api.github.com/repos/"+repo+"/commits/"+sha, &commit)
	if err != nil {
		return 0, err
	}
	if st != http.StatusOK {
		return 0, xerrors.New(fmt.Sprintf("commits/%s: %d", sha, st))
	}
	return daysSince(commit.Commit.Committer.Date), nil
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

// applyOrCheck は lockfile を SSOT に uses: を固定する。dryRun=true は書き換えず drift を非ゼロ終了で報告する。
func applyOrCheck(root string, files []string, dryRun bool) {
	lock, err := readLock(filepath.Join(root, lockFile))
	if err != nil {
		log.Fatalf("❌ read lockfile（先に resolve を実行してください）: %v", err)
	}
	var missing, drifted []string
	changed := 0
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			log.Fatalf("❌ read %s: %v", f, err)
		}
		out, miss := rewritePins(string(data), lock)
		missing = append(missing, miss...)
		if out == string(data) {
			continue
		}
		if dryRun {
			drifted = append(drifted, rel(root, f))
			continue
		}
		if err := os.WriteFile(f, []byte(out), filePerm); err != nil { //nolint:gosec // workflow ファイル権限
			log.Fatalf("❌ write %s: %v", f, err)
		}
		changed++
		log.Printf("  updated %s", rel(root, f))
	}
	report(missing, drifted, dryRun, changed)
}

func report(missing, drifted []string, dryRun bool, changed int) {
	if len(missing) > 0 {
		sort.Strings(missing)
		log.Fatalf("❌ lockfile に未登録の参照があります（make pin-actions-resolve を実行してください）: %s",
			strings.Join(uniq(missing), ", "))
	}
	if !dryRun {
		log.Printf("✅ %d ファイルを固定しました", changed)
		return
	}
	if len(drifted) > 0 {
		sort.Strings(drifted)
		log.Fatalf("❌ 未固定/古い参照があります（make pin-actions-resolve && make pin-actions-apply してコミットしてください）: %s",
			strings.Join(drifted, ", "))
	}
	log.Printf("✅ 全アクションが lockfile 通りに固定されています")
}

// resolveSHA は owner/repo の tag/branch を commit SHA へ解決する。annotated tag は ^{} で deref する。
// 出力パースと ref 優先選択は selectSHA へ委譲し、本関数は git ls-remote の実行のみを担う。
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
		return "", xerrors.New(fmt.Sprintf("ref %q が見つかりません", tag))
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
// nil（成功）と「ファイル不在」（初回 resolve）のみ true。それ以外（権限・破損等）は fail-close 対象。
func isIgnorableLockErr(err error) bool {
	return err == nil || xerrors.Is(err, os.ErrNotExist)
}

func readLock(path string) (map[string]string, error) {
	f, err := os.Open(path) //nolint:gosec // path is constructed from cwd + literal filename
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	lock := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := lockRe.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			lock[m[1]] = m[2]
		}
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
