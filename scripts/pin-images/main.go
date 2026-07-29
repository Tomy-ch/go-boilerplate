// Package main は Dockerfile の `FROM` base image と docker compose の `image:` を
// 不変の digest へ固定するツール。
//
//	resolve: docker/*/Dockerfile の FROM と docker-compose*.yaml の image: を走査し、
//	         image:tag を registry の現在 digest へ解決して lockfile (docker/images-pin.toml) へ書き出す。
//	apply:   lockfile を SSOT に各参照を `image:tag@sha256:...` へ固定する（FROM は AS stage を保持）。
//	check:   apply と同じ判定を書き換えなしで行い、未固定/未登録/drift があれば非ゼロ終了する（CI / hook 用）。
//
// tag は版の SSOT として FROM 行に残し、digest を lock 側で管理する（tag 再割り当て攻撃の遮断）。
//
// supply-chain cooldown: --min-age-days 未満（＝公開直後）の digest は採用しない。
// mutable tag は「N 日前に何を指していたか」を registry へ問えないため、step-back 先は
// git tag のような版履歴ではなく自分の前回 lock。前回 lock が無い初回は退行先が無く、
// 出来立ての未検証 digest を掴む危険があるため、resolve は tag のまま残さず非ゼロ終了する
// （緊急ブートストラップのみ --min-age-days=0 で明示採用）。
//
// apply/check は fail-closed：管理対象の FROM は image:tag@sha256:... へ固定され、かつ lock(SSOT)
// に登録済みでなければならない。lock 未登録の image は「未登録」、digest 無し（tag のみ）は
// 「未固定」として非ゼロ終了する（tag のみ運用は一切許容しない）。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
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
	lockFile        = "docker/images-pin.toml"
	filePerm        = 0o644
	minArgs         = 2 // プログラム名 + サブコマンド
	inspectTimeout  = 60 * time.Second
	hoursPerDay     = 24
	createdTimeCols = 3 // "2006-01-02 15:04:05.000 +0000 UTC" を最初の 3 フィールドで再構成
)

var (
	// FROM [--platform=...] <ref> [AS <stage>]
	fromRe = regexp.MustCompile(`(?m)^(FROM[ \t]+)(?:--platform=\S+[ \t]+)?(\S+)([ \t]+(?i:AS)[ \t]+\S+)?[ \t]*$`)
	// docker compose service の `image: <ref>`。FROM と同じ prefix/ref/suffix の 3 グループ構成に
	// して rewritePins を共用する。suffix は末尾空白と行末コメントを取り込み保持する。
	composeImageRe = regexp.MustCompile(`(?m)^([ \t]+image:[ \t]+)(\S+)([ \t]*(?:#.*)?)$`)
	// lockfile 行: "image:tag" = "sha256:..."
	lockRe   = regexp.MustCompile(`^"([^"]+)"\s*=\s*"(sha256:[0-9a-f]+)"`)
	digestRe = regexp.MustCompile(`(?m)^Digest:[ \t]+(sha256:[0-9a-f]+)`)
)

var (
	// errDigestUnparsable は、inspect 出力から Digest 行を解析できなかった場合のエラー。
	errDigestUnparsable = xerrors.New("Digest 行を解析できません")
	// errCreatedUnparsable は、inspect 出力から image config の created を解析できなかった場合のエラー。
	errCreatedUnparsable = xerrors.New("created を解析できません")
)

// target は走査対象のファイルと、その参照行を捕捉する正規表現（prefix/ref/suffix の 3 グループ）。
type target struct {
	path string
	re   *regexp.Regexp
}

// imageRef は FROM が参照する registry image 1 件。key は image:tag。
type imageRef struct {
	image string // 例: golang, nginx, ghcr.io/foo/bar
	tag   string // 例: 1.26.5-alpine
}

func (r imageRef) key() string { return r.image + ":" + r.tag }

func main() {
	log.SetFlags(0)
	if len(os.Args) < minArgs {
		log.Fatalf("❌ usage: pin-images <resolve|apply|check>")
	}
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ getwd: %v", err)
	}
	targets, err := targetFiles(root)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	switch os.Args[1] {
	case "resolve":
		fs := flag.NewFlagSet("resolve", flag.ExitOnError)
		minAge := fs.Int("min-age-days", 0, "N 日未満の新しすぎる digest は quarantine（0 で無効）")
		_ = fs.Parse(os.Args[2:])
		resolve(root, targets, *minAge)
	case "apply":
		applyOrCheck(root, targets, false)
	case "check":
		applyOrCheck(root, targets, true)
	default:
		log.Fatalf("❌ usage: pin-images <resolve|apply|check>")
	}
}

func targetFiles(root string) ([]target, error) {
	dockerfiles, err := filepath.Glob(filepath.Join(root, "docker/*/Dockerfile"))
	if err != nil {
		return nil, xerrors.Wrap(err, "glob docker/*/Dockerfile")
	}
	compose, err := filepath.Glob(filepath.Join(root, "docker-compose*.yaml"))
	if err != nil {
		return nil, xerrors.Wrap(err, "glob docker-compose*.yaml")
	}
	var targets []target
	for _, f := range dockerfiles {
		targets = append(targets, target{path: f, re: fromRe})
	}
	for _, f := range compose {
		targets = append(targets, target{path: f, re: composeImageRe})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].path < targets[j].path })
	return targets, nil
}

// parseRef は FROM / compose image の ref を image:tag へ分解する。第2戻り値が false なら対象外
// （tag 無し＝ビルドステージ参照 / scratch、あるいは registry port を tag と誤認する形）。
func parseRef(ref string) (imageRef, bool) {
	name, _, _ := strings.Cut(ref, "@") // 既存の @digest を捨てる
	image, tag, ok := lastColonSplit(name)
	if !ok || tag == "" || strings.Contains(tag, "/") {
		return imageRef{}, false // tag が無い / 最後の ':' が registry:port だった
	}
	return imageRef{image: image, tag: tag}, true
}

// lastColonSplit は最後の ':' で分割する（tag に ':' は含まれない前提）。
func lastColonSplit(s string) (string, string, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func resolve(root string, targets []target, minAgeDays int) {
	keys := collectKeys(targets)
	existing, _ := readLock(filepath.Join(root, lockFile)) // 無ければ空

	ctx := context.Background()
	lock := map[string]string{}
	var notes, skipped []string
	for k, r := range keys {
		digest, err := resolveDigest(ctx, r.key())
		if err != nil {
			log.Fatalf("❌ resolve %s: %v", k, err)
		}
		use, note := quarantine(ctx, r, k, digest, minAgeDays, existing)
		if note != "" {
			notes = append(notes, note)
		}
		if use == "" {
			skipped = append(skipped, k)
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

	// 退行先の無い出来立て image（fresh + 既存ピン無し）は tag のまま残さず失敗させる。
	// tag のみ運用を許すと未検証 digest を掴むため、check も同状態を弾く（fail-closed）。
	if len(skipped) > 0 {
		sort.Strings(skipped)
		log.Fatalf("❌ 退行先の無い出来立て image は採用できません（%d 日未満・既存ピン無し）。"+
			"aged 後に再実行するか、緊急時のみ --min-age-days=0 で明示採用してください: %s",
			minAgeDays, strings.Join(skipped, ", "))
	}
}

func collectKeys(targets []target) map[string]imageRef {
	keys := map[string]imageRef{}
	for _, t := range targets {
		data, err := os.ReadFile(t.path)
		if err != nil {
			log.Fatalf("❌ read %s: %v", t.path, err)
		}
		for _, m := range t.re.FindAllStringSubmatch(string(data), -1) {
			if r, ok := parseRef(m[2]); ok {
				keys[r.key()] = r
			}
		}
	}
	return keys
}

// quarantine は minAgeDays 未満の新しすぎる digest を採用しない。第1戻り値 "" は skip。
func quarantine(ctx context.Context, r imageRef, key, candidate string, minAgeDays int, existing map[string]string) (string, string) {
	if minAgeDays <= 0 {
		return candidate, ""
	}
	age, err := digestAgeDays(ctx, r.key())
	if err != nil {
		log.Fatalf("❌ age %s: %v", key, err)
	}
	if age >= minAgeDays {
		return candidate, ""
	}
	if prev, ok := existing[key]; ok {
		return prev, fmt.Sprintf("%s: 現 digest が %d 日 (<%d) のため既存ピンを維持", key, age, minAgeDays)
	}
	return "", fmt.Sprintf("%s: 現 digest が %d 日 (<%d)・既存ピン無しのため tag のまま skip", key, age, minAgeDays)
}

// resolveDigest は image:tag の現在の index digest を registry から取得する。
func resolveDigest(ctx context.Context, ref string) (string, error) {
	out, err := inspect(ctx, ref)
	if err != nil {
		return "", err
	}
	m := digestRe.FindStringSubmatch(out)
	if m == nil {
		return "", xerrors.Wrap(errDigestUnparsable, ref)
	}
	return m[1], nil
}

// digestAgeDays は image:tag が指す digest の image config created から経過日数を返す。
// 公式イメージのビルド日は publish 日にほぼ一致する。マルチアーキの場合は最も古い created を採る。
func digestAgeDays(ctx context.Context, ref string) (int, error) {
	created, err := earliestCreated(ctx, ref)
	if err != nil {
		return 0, err
	}
	return int(time.Since(created).Hours() / hoursPerDay), nil
}

func earliestCreated(ctx context.Context, ref string) (time.Time, error) {
	// マルチアーキ (index) は .Image が map、単一アーキは struct。前者→後者でフォールバックする。
	// inspect 自体のエラー（429 等）は握り潰さず伝播し、テンプレート不一致のときだけ次を試す。
	var lastErr error
	for _, tmpl := range []string{
		`{{ range .Image }}{{ println .Created }}{{ end }}`,
		`{{ println .Image.Created }}`,
	} {
		out, err := inspect(ctx, ref, "--format", tmpl)
		if err != nil {
			lastErr = err
			continue
		}
		if t, ok := minCreated(out); ok {
			return t, nil
		}
	}
	if lastErr != nil {
		return time.Time{}, lastErr
	}
	return time.Time{}, xerrors.Wrap(errCreatedUnparsable, ref)
}

// minCreated は inspect 出力の各行（"2006-01-02 15:04:05.9 +0000 UTC"）から最古の時刻を返す。
func minCreated(out string) (time.Time, bool) {
	var earliest time.Time
	found := false
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < createdTimeCols {
			continue
		}
		// "date time zone" の 3 フィールドだけを RFC 化して parse（末尾 "UTC" 等は捨てる）。
		t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700",
			strings.Join(fields[:createdTimeCols], " "))
		if err != nil {
			continue
		}
		if !found || t.Before(earliest) {
			earliest, found = t, true
		}
	}
	return earliest, found
}

func inspect(ctx context.Context, ref string, extra ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, inspectTimeout)
	defer cancel()
	args := append([]string{"buildx", "imagetools", "inspect", ref}, extra...)
	cmd := exec.CommandContext(cctx, "docker", args...) //nolint:gosec // ref は Dockerfile 由来
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", xerrors.Wrap(err, "docker buildx imagetools inspect "+ref+": "+strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// rewritePins は lock を元に FROM を digest 固定した内容と、lock 未登録の image キー一覧を返す。
// lock に無い image は「未登録」として報告し、行は書き換えない（digest は剥がさない）。
func rewritePins(data string, re *regexp.Regexp, lock map[string]string) (string, []string) {
	var missing []string
	out := re.ReplaceAllStringFunc(data, func(line string) string {
		m := re.FindStringSubmatch(line)
		r, ok := parseRef(m[2])
		if !ok {
			return line
		}
		digest, found := lock[r.key()]
		if !found {
			missing = append(missing, r.key())
			return line
		}
		return m[1] + r.key() + "@" + digest + m[3]
	})
	return out, missing
}

// applyOrCheck は lockfile を SSOT に FROM を digest 固定する。dryRun=true は書き換えず
// 未固定/未登録/drift を非ゼロ終了で報告する。tag のみへ戻す正規化はしない（fail-closed）。
func applyOrCheck(root string, targets []target, dryRun bool) {
	lock, err := readLock(filepath.Join(root, lockFile))
	if err != nil {
		log.Fatalf("❌ read lockfile（先に make pin-images-resolve を実行してください）: %v", err)
	}
	var missing, drifted []string
	changed := 0
	for _, t := range targets {
		data, err := os.ReadFile(t.path)
		if err != nil {
			log.Fatalf("❌ read %s: %v", t.path, err)
		}
		out, miss := rewritePins(string(data), t.re, lock)
		missing = append(missing, miss...)
		if out == string(data) {
			continue
		}
		if dryRun {
			drifted = append(drifted, rel(root, t.path))
			continue
		}
		if err := os.WriteFile(t.path, []byte(out), filePerm); err != nil { //nolint:gosec // 管理対象ファイル権限
			log.Fatalf("❌ write %s: %v", t.path, err)
		}
		changed++
		log.Printf("  updated %s", rel(root, t.path))
	}
	report(missing, drifted, dryRun, changed)
}

func report(missing, drifted []string, dryRun bool, changed int) {
	if len(missing) > 0 {
		sort.Strings(missing)
		log.Fatalf("❌ lockfile に未登録の base image があります（make pin-images-resolve を実行してください）: %s",
			strings.Join(uniq(missing), ", "))
	}
	if !dryRun {
		log.Printf("✅ %d ファイルを固定しました", changed)
		return
	}
	if len(drifted) > 0 {
		sort.Strings(drifted)
		log.Fatalf("❌ image 参照が未固定か lockfile と不一致です（make pin-images-resolve && make pin-images-apply してコミットしてください）: %s",
			strings.Join(drifted, ", "))
	}
	log.Printf("✅ 全 base image が lockfile 通りに固定されています")
}

func writeLock(path string, lock map[string]string) error {
	keys := make([]string, 0, len(lock))
	for k := range lock {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# Docker base image / docker compose image の pin 対象 digest（SSOT）。\n")
	b.WriteString("# make pin-images-resolve で解決・make pin-images-apply で Dockerfile / docker-compose へ反映する。\n")
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

// uniq はソート済みスライスから隣接重複を除く。
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
