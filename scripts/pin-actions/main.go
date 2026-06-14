// pin-actions は GitHub Actions の `uses:` 参照を不変の commit SHA へ固定するツール。
//
//	resolve: .github/workflows/** と .github/actions/** の外部アクション参照を走査し、
//	         tag/version を git ls-remote で SHA へ解決して lockfile (.github/actions-pin.toml) へ書き出す。
//	apply:   lockfile を SSOT に各ファイルの `uses: owner/repo[/sub]@<ref>` を
//	         `uses: owner/repo[/sub]@<sha> # <ref>` へ置換する。
//
// 既に固定済み (`@<sha> # <ref>`) の行は、コメントの <ref> を版として再解決するため idempotent。
// ローカル参照 (`uses: ./...`) は @ を持たないため対象外。
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	lockFile        = ".github/actions-pin.toml"
	filePerm        = 0o644
	minArgs         = 2 // プログラム名 + サブコマンド
	repoSegments    = 2 // owner/repo
	lsRemoteCols    = 2 // <sha>\t<refname>
	lsRemoteTimeout = 30 * time.Second
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
		log.Fatalf("❌ usage: pin-actions <resolve|apply>")
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
		resolve(root, files)
	case "apply":
		apply(root, files)
	default:
		log.Fatalf("❌ usage: pin-actions <resolve|apply>")
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
			return nil, fmt.Errorf("glob %s: %w", pat, err)
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	return files, nil
}

func resolve(root string, files []string) {
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

	ctx := context.Background()
	lock := map[string]string{}
	for k, r := range keys {
		sha, err := resolveSHA(ctx, r.repo, r.tag)
		if err != nil {
			log.Fatalf("❌ resolve %s: %v", k, err)
		}
		lock[k] = sha
		log.Printf("  %s -> %s", k, sha)
	}

	if err := writeLock(filepath.Join(root, lockFile), lock); err != nil {
		log.Fatalf("❌ write lockfile: %v", err)
	}
	log.Printf("✅ %s に %d 件を書き出しました", lockFile, len(lock))
}

func apply(root string, files []string) {
	lock, err := readLock(filepath.Join(root, lockFile))
	if err != nil {
		log.Fatalf("❌ read lockfile（先に resolve を実行してください）: %v", err)
	}

	var missing []string
	changed := 0
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			log.Fatalf("❌ read %s: %v", f, err)
		}
		out := usesRe.ReplaceAllStringFunc(string(data), func(line string) string {
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
		if out == string(data) {
			continue
		}
		if err := os.WriteFile(f, []byte(out), filePerm); err != nil { //nolint:gosec // workflow ファイル権限
			log.Fatalf("❌ write %s: %v", f, err)
		}
		changed++
		log.Printf("  updated %s", rel(root, f))
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		log.Fatalf("❌ lockfile に未登録の参照があります（resolve を再実行してください）: %s",
			strings.Join(uniq(missing), ", "))
	}
	log.Printf("✅ %d ファイルを固定しました", changed)
}

// resolveSHA は owner/repo の tag/branch を commit SHA へ解決する。annotated tag は ^{} で deref する。
func resolveSHA(ctx context.Context, repo, tag string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, lsRemoteTimeout)
	defer cancel()
	url := "https://github.com/" + repo
	out, err := exec.CommandContext(cctx, "git", "ls-remote", url, tag, tag+"^{}").Output() //nolint:gosec // 参照名は workflow 由来
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	var tagSHA, derefSHA, headSHA string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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
		return "", fmt.Errorf("ref %q が見つかりません", tag)
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
