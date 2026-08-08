// Package main は harden-runner の allowed-endpoints を SSOT から生成・検証するツール。
//
//	apply: .github/egress.toml を元に各 workflow の allowed-endpoints ブロックを書き換える。
//	check: apply と同じ判定を書き換えなしで行い、ドリフトがあれば非ゼロ終了する（CI / hook 用）。
//
// 詳細は .github/workflows/README.md の Runner Hardening 節を参照。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go-boilerplate/pkg/xerrors"
)

const (
	ssotFile = ".github/egress.toml"
	filePerm = 0o644
	// policyAudit は allowed-endpoints を持たない唯一の egress-policy。
	policyAudit = "audit"
	policyBlock = "block"
	// blockIndentStep は allowed-endpoints キーに対するホスト行の相対字下げ。
	blockIndentStep = 2
)

var (
	sectionRe      = regexp.MustCompile(`^\[(class|job)\.(?:"([^"]+)"|([A-Za-z0-9_-]+))\]$`)
	scalarRe       = regexp.MustCompile(`^(\w+)\s*=\s*"([^"]*)"$`)
	arrayHeadRe    = regexp.MustCompile(`^(\w+)\s*=\s*\[(.*)$`)
	quotedRe       = regexp.MustCompile(`"([^"]*)"`)
	jobIDRe        = regexp.MustCompile(`^  ([A-Za-z_][A-Za-z0-9_-]*):\s*$`)
	egressPolicyRe = regexp.MustCompile(`^\s*egress-policy:\s*(\S+)\s*$`)
	endpointsKeyRe = regexp.MustCompile(`^(\s*)allowed-endpoints:\s*>\s*$`)
)

var (
	// errUsage は、サブコマンドが無いか未知の場合のエラー。
	errUsage = xerrors.New("usage: egress <apply|check>")
	// errSSOTSyntax は、SSOT に解釈できない行があった場合のエラー。
	errSSOTSyntax = xerrors.New("SSOT に解釈できない行があります")
	// errSSOTDuplicate は、SSOT にセクション / キー / ホストの重複があった場合のエラー。
	errSSOTDuplicate = xerrors.New("SSOT に重複があります")
	// errSSOTUnknownClass は、未定義のクラスを参照した場合のエラー。
	errSSOTUnknownClass = xerrors.New("SSOT が未定義のクラスを参照しています")
	// errSSOTClassCycle は、クラスの includes が循環している場合のエラー。
	errSSOTClassCycle = xerrors.New("SSOT のクラス継承が循環しています")
	// errSSOTBaseClass は、base クラスを classes に明示した場合のエラー。
	errSSOTBaseClass = xerrors.New("base クラスは全ジョブへ暗黙に適用されるため classes に書けません")
	// errSSOTPolicy は、egress_policy の値が不正、または audit ジョブが許可リストを持つ場合のエラー。
	errSSOTPolicy = xerrors.New("SSOT の egress_policy が不正です")
	// errWorkflowJobless は、どのジョブにも属さない harden-runner 記述を検出した場合のエラー。
	errWorkflowJobless = xerrors.New("ジョブに属さない harden-runner の記述があります")
	// errBlockComment は、allowed-endpoints ブロックにホスト以外の行があった場合のエラー。
	errBlockComment = xerrors.New("allowed-endpoints ブロックにホスト以外の行があります")
	// errPolicyMismatch は、workflow の egress-policy と SSOT の宣言が食い違う場合のエラー。
	errPolicyMismatch = xerrors.New("workflow の egress-policy と SSOT の宣言が食い違います")
	// errJobMissing は、workflow のジョブが SSOT に未登録の場合のエラー。
	errJobMissing = xerrors.New("SSOT に未登録のジョブがあります")
	// errJobOrphan は、SSOT にあるがどの workflow にも存在しないジョブがあった場合のエラー。
	errJobOrphan = xerrors.New("SSOT に対応する workflow が無いジョブがあります")
	// errEgressDrift は、check で SSOT との差分を検出した場合のエラー。
	errEgressDrift = xerrors.New("allowed-endpoints が SSOT からずれています")
)

// class は能力クラス 1 件。includes は継承元クラス、hosts は自クラスが足す host:port。
type class struct {
	includes []string
	hosts    []string
}

// jobSpec は 1 ジョブの宣言。key は `<workflow ファイル名>:<job id>`。
type jobSpec struct {
	classes []string
	extra   []string
	policy  string
}

// ssot は .github/egress.toml の内容。order は宣言順（出力の並びを SSOT が決める）。
type ssot struct {
	classes   map[string]*class
	jobs      map[string]*jobSpec
	jobOrder  []string
	baseHosts []string
}

// orderedHosts は初出順を保ったまま重複を落とすホスト集合。
type orderedHosts struct {
	list []string
	seen map[string]bool
}

// block は workflow 内の allowed-endpoints ブロック 1 件の位置と中身。
type block struct {
	jobKey string
	// keyLine は `allowed-endpoints: >` の行番号（0 起点）。
	keyLine int
	// endLine はホスト行の終端（0 起点・排他）。
	endLine int
	indent  int
	hosts   []string
}

// main はエラーを終了コードへ変換するだけに留め、判断は run が持つ。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:], os.Getwd); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

// run はサブコマンドを解釈して apply / check へ振り分ける。wd は走査の基点の取得手段。
func run(args []string, wd func() (string, error)) error {
	if len(args) != 1 {
		return errUsage
	}
	root, err := wd()
	if err != nil {
		return xerrors.Wrap(err, "getwd")
	}
	switch args[0] {
	case "apply":
		return applyOrCheck(root, false)
	case "check":
		return applyOrCheck(root, true)
	default:
		return errUsage
	}
}

// parseSSOT は SSOT の本文をクラス定義とジョブ宣言へ分解する。
func parseSSOT(data string) (*ssot, error) {
	s := &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}}
	var curClass *class
	var curJob *jobSpec
	var seen map[string]bool

	lines := strings.Split(data, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(stripComment(lines[i]))
		if line == "" {
			continue
		}
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			name := m[2] + m[3]
			if err := s.openSection(m[1], name); err != nil {
				return nil, err
			}
			curClass, curJob = s.classes[name], s.jobs[name]
			seen = map[string]bool{}
			continue
		}
		if curClass == nil && curJob == nil {
			return nil, xerrors.Wrap(errSSOTSyntax, fmt.Sprintf("%d 行目: %q（セクションの外に値があります）", i+1, line))
		}
		key, values, next, err := parseAssignment(lines, i)
		if err != nil {
			return nil, err
		}
		i = next
		if seen[key] {
			return nil, xerrors.Wrap(errSSOTDuplicate, fmt.Sprintf("%d 行目: キー %q", i+1, key))
		}
		seen[key] = true
		if err := assign(curClass, curJob, key, values); err != nil {
			return nil, xerrors.Wrap(err, fmt.Sprintf("%d 行目", i+1))
		}
	}
	return s, s.validate()
}

// stripComment は行末コメントを落とす。値のクオート内に `#` は現れない前提。
func stripComment(line string) string {
	if before, _, ok := strings.Cut(line, "#"); ok {
		return before
	}
	return line
}

// openSection は新しい `[class.x]` / `[job."x"]` を登録する。既出なら重複エラー。
func (s *ssot) openSection(kind, name string) error {
	if _, dup := s.classes[name]; dup && kind == "class" {
		return xerrors.Wrap(errSSOTDuplicate, "class "+name)
	}
	if _, dup := s.jobs[name]; dup && kind == "job" {
		return xerrors.Wrap(errSSOTDuplicate, "job "+name)
	}
	if kind == "class" {
		s.classes[name] = &class{}
		return nil
	}
	s.jobs[name] = &jobSpec{}
	s.jobOrder = append(s.jobOrder, name)
	return nil
}

// parseAssignment は lines[i] から始まる代入を読み、キー・値・消費した最終行を返す。
// 配列は `]` が現れるまで後続行を取り込む。
func parseAssignment(lines []string, i int) (string, []string, int, error) {
	line := strings.TrimSpace(stripComment(lines[i]))
	if m := scalarRe.FindStringSubmatch(line); m != nil {
		return m[1], []string{m[2]}, i, nil
	}
	m := arrayHeadRe.FindStringSubmatch(line)
	if m == nil {
		return "", nil, 0, xerrors.Wrap(errSSOTSyntax, fmt.Sprintf("%d 行目: %q", i+1, line))
	}
	key, rest := m[1], m[2]
	var values []string
	for {
		closed := strings.Contains(rest, "]")
		for _, q := range quotedRe.FindAllStringSubmatch(rest, -1) {
			values = append(values, q[1])
		}
		if closed {
			return key, values, i, nil
		}
		i++
		if i >= len(lines) {
			return "", nil, 0, xerrors.Wrap(errSSOTSyntax, "配列が閉じられていません: "+key)
		}
		rest = stripComment(lines[i])
	}
}

// assign は読み取ったキーを、開いているセクションのフィールドへ載せる。
func assign(c *class, j *jobSpec, key string, values []string) error {
	if err := requireUnique(values); err != nil {
		return xerrors.Wrap(err, key)
	}
	switch {
	case c != nil && key == "includes":
		c.includes = values
	case c != nil && key == "hosts":
		c.hosts = values
	case j != nil && key == "classes":
		j.classes = values
	case j != nil && key == "extra":
		j.extra = values
	case j != nil && key == "egress_policy" && len(values) == 1:
		j.policy = values[0]
	default:
		return xerrors.Wrap(errSSOTSyntax, "未知のキー "+key)
	}
	return nil
}

// requireUnique は同一リスト内の重複を弾く。重複は「どちらを直せばよいか」が読めなくなる。
func requireUnique(values []string) error {
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return xerrors.Wrap(errSSOTDuplicate, v)
		}
		seen[v] = true
	}
	return nil
}

// validate は SSOT 単体で判定できる不整合を返す。
func (s *ssot) validate() error {
	if base, ok := s.classes["base"]; ok {
		s.baseHosts = base.hosts
	}
	for name, c := range s.classes {
		for _, inc := range c.includes {
			if _, ok := s.classes[inc]; !ok {
				return xerrors.Wrap(errSSOTUnknownClass, name+" includes "+inc)
			}
		}
	}
	for _, key := range s.jobOrder {
		if err := s.validateJob(key, s.jobs[key]); err != nil {
			return err
		}
	}
	return nil
}

// validateJob は 1 ジョブの宣言の整合を返す。
func (s *ssot) validateJob(key string, j *jobSpec) error {
	switch j.policy {
	case "", policyBlock:
		j.policy = policyBlock
	case policyAudit:
		if len(j.classes) > 0 || len(j.extra) > 0 {
			return xerrors.Wrap(errSSOTPolicy, key+": audit は allowed-endpoints を持ちません")
		}
	default:
		return xerrors.Wrap(errSSOTPolicy, key+": "+j.policy)
	}
	for _, cn := range j.classes {
		if cn == "base" {
			return xerrors.Wrap(errSSOTBaseClass, key)
		}
		if _, ok := s.classes[cn]; !ok {
			return xerrors.Wrap(errSSOTUnknownClass, key+" classes "+cn)
		}
	}
	_, err := s.hostsFor(key)
	return err
}

// hostsFor は 1 ジョブの許可リストを算出する。base → classes（includes 展開込み）→ extra の順で、
// 重複は初出を残す。並びを SSOT の宣言順に固定しているため、生成物の差分はレビューで読める。
func (s *ssot) hostsFor(key string) ([]string, error) {
	j := s.jobs[key]
	if j.policy == policyAudit {
		return nil, nil
	}
	out := &orderedHosts{seen: map[string]bool{}}
	out.add(s.baseHosts)
	for _, cn := range j.classes {
		if err := s.expandClass(cn, out, map[string]bool{}); err != nil {
			return nil, xerrors.Wrap(err, key)
		}
	}
	out.add(j.extra)
	return out.list, nil
}

// expandClass は includes を先に展開してからクラス自身の hosts を足す。visiting は循環検出用。
func (s *ssot) expandClass(name string, out *orderedHosts, visiting map[string]bool) error {
	if visiting[name] {
		return xerrors.Wrap(errSSOTClassCycle, name)
	}
	visiting[name] = true
	defer delete(visiting, name)

	c, ok := s.classes[name]
	if !ok {
		return xerrors.Wrap(errSSOTUnknownClass, name)
	}
	for _, inc := range c.includes {
		if err := s.expandClass(inc, out, visiting); err != nil {
			return err
		}
	}
	out.add(c.hosts)
	return nil
}

func (o *orderedHosts) add(hosts []string) {
	for _, h := range hosts {
		if o.seen[h] {
			continue
		}
		o.seen[h] = true
		o.list = append(o.list, h)
	}
}

// scanWorkflow は 1 ファイルの harden-runner 記述を、ジョブ単位の egress-policy と
// allowed-endpoints ブロックへ分解する。name は workflow のファイル名。
func scanWorkflow(name string, lines []string) (map[string]string, []block, error) {
	policies := map[string]string{}
	var blocks []block
	jobs := workflowJobs(lines)

	for i := 0; i < len(lines); i++ {
		if m := egressPolicyRe.FindStringSubmatch(lines[i]); m != nil {
			if jobs[i] == "" {
				return nil, nil, xerrors.Wrap(errWorkflowJobless, fmt.Sprintf("%s %d 行目", name, i+1))
			}
			policies[name+":"+jobs[i]] = m[1]
			continue
		}
		m := endpointsKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if jobs[i] == "" {
			return nil, nil, xerrors.Wrap(errWorkflowJobless, fmt.Sprintf("%s %d 行目", name, i+1))
		}
		b, err := readBlock(name, lines, i, len(m[1]))
		if err != nil {
			return nil, nil, err
		}
		b.jobKey = name + ":" + jobs[i]
		blocks = append(blocks, b)
		i = b.endLine - 1
	}
	return policies, blocks, nil
}

// workflowJobs は各行が属する `jobs:` 直下の job id を行番号順に返す。ジョブの外は空文字。
func workflowJobs(lines []string) []string {
	out := make([]string, len(lines))
	inJobs, job := false, ""
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "#"):
		case line != "" && !strings.HasPrefix(line, " "):
			inJobs, job = strings.HasPrefix(line, "jobs:"), ""
		case inJobs:
			if m := jobIDRe.FindStringSubmatch(line); m != nil {
				job = m[1]
			}
		}
		out[i] = job
	}
	return out
}

// readBlock は keyLine から始まる折り畳みスカラーのホスト行を読む。
// ホスト 1 個以外の行（コメント・複数トークン）は握り潰さず fail-close する。
// 黙って落とすと、生成物がそのコメントを消したことに誰も気づけない。
func readBlock(name string, lines []string, keyLine, indent int) (block, error) {
	b := block{keyLine: keyLine, endLine: keyLine + 1, indent: indent}
	for j := keyLine + 1; j < len(lines); j++ {
		line := lines[j]
		if strings.TrimSpace(line) == "" {
			break
		}
		if len(line)-len(strings.TrimLeft(line, " ")) <= indent {
			break
		}
		host := strings.TrimSpace(line)
		if strings.HasPrefix(host, "#") || strings.ContainsAny(host, " \t") {
			return block{}, xerrors.Wrap(errBlockComment, fmt.Sprintf("%s %d 行目: %q", name, j+1, host))
		}
		b.hosts = append(b.hosts, host)
		b.endLine = j + 1
	}
	return b, nil
}

// renderBlock はホスト行を workflow へ書き戻す形へ整える。
func renderBlock(hosts []string, indent int) []string {
	pad := strings.Repeat(" ", indent+blockIndentStep)
	out := make([]string, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, pad+h)
	}
	return out
}

// applyOrCheck は SSOT を正本に allowed-endpoints を書き換える。dryRun=true は差分を非ゼロ終了で報告する。
func applyOrCheck(root string, dryRun bool) error {
	raw, err := os.ReadFile(filepath.Join(root, ssotFile)) //nolint:gosec // path is cwd + literal filename
	if err != nil {
		return xerrors.Wrap(err, "read "+ssotFile)
	}
	s, err := parseSSOT(string(raw))
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yaml"))
	if err != nil {
		return xerrors.Wrap(err, "glob workflows")
	}
	sort.Strings(files)

	changes, used, err := s.planFiles(files)
	if err != nil {
		return err
	}
	if orphans := orphanJobs(s, used); len(orphans) > 0 {
		return xerrors.Wrap(errJobOrphan, strings.Join(orphans, ", ")+"（"+ssotFile+" の該当エントリを削除してください）")
	}
	return writeChanges(root, changes, dryRun)
}

// planFiles は全 workflow を読み切り、書き換え後の内容と参照済みジョブキーを確定させる。
// 1 ファイルずつ書きながら進むと、後半の不整合で中断したときに書き換え済みの中途半端な作業ツリーが残る。
func (s *ssot) planFiles(files []string) (map[string]string, map[string]bool, error) {
	changes := map[string]string{}
	used := map[string]bool{}
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // path from cwd glob
		if err != nil {
			return nil, nil, xerrors.Wrap(err, "read "+f)
		}
		out, err := s.rewriteFile(filepath.Base(f), string(data), used)
		if err != nil {
			return nil, nil, err
		}
		if out != string(data) {
			changes[f] = out
		}
	}
	return changes, used, nil
}

// rewriteFile は 1 ファイルの allowed-endpoints ブロックを SSOT 由来の内容へ差し替えた本文を返す。
func (s *ssot) rewriteFile(name, data string, used map[string]bool) (string, error) {
	lines := strings.Split(data, "\n")
	policies, blocks, err := scanWorkflow(name, lines)
	if err != nil {
		return "", err
	}
	byJob := map[string]bool{}
	for _, b := range blocks {
		byJob[b.jobKey] = true
	}
	for key, policy := range policies {
		if err := s.checkPolicy(key, policy, byJob[key]); err != nil {
			return "", err
		}
		used[key] = true
	}
	// 後ろから差し替えて、先行ブロックの行番号を保つ。
	for _, v := range slices.Backward(blocks) {
		b := v
		hosts, err := s.hostsFor(b.jobKey)
		if err != nil {
			return "", err
		}
		lines = append(lines[:b.keyLine+1:b.keyLine+1],
			append(renderBlock(hosts, b.indent), lines[b.endLine:]...)...)
	}
	return strings.Join(lines, "\n"), nil
}

// checkPolicy は workflow 側の egress-policy と SSOT の宣言、ブロックの在否の 3 者が揃うことを検証する。
func (s *ssot) checkPolicy(key, policy string, hasBlock bool) error {
	j, ok := s.jobs[key]
	if !ok {
		return xerrors.Wrap(errJobMissing, key+"（"+ssotFile+" へ追加してください）")
	}
	if j.policy != policy {
		return xerrors.Wrap(errPolicyMismatch, fmt.Sprintf("%s: workflow=%s SSOT=%s", key, policy, j.policy))
	}
	if hasBlock != (policy == policyBlock) {
		return xerrors.Wrap(errPolicyMismatch,
			fmt.Sprintf("%s: egress-policy=%s に対して allowed-endpoints の在否が合いません", key, policy))
	}
	return nil
}

// orphanJobs は SSOT にあるがどの workflow にも現れないジョブキーを返す。
// 孤児を残すと SSOT が現用インベントリの鏡でなくなる。
func orphanJobs(s *ssot, used map[string]bool) []string {
	var orphans []string
	for _, key := range s.jobOrder {
		if !used[key] {
			orphans = append(orphans, key)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// writeChanges は確定した書き換えを反映する。dryRun=true は差分の一覧を非ゼロ終了で報告する。
func writeChanges(root string, changes map[string]string, dryRun bool) error {
	paths := make([]string, 0, len(changes))
	for f := range changes {
		paths = append(paths, f)
	}
	sort.Strings(paths)

	if dryRun {
		if len(paths) > 0 {
			rels := make([]string, 0, len(paths))
			for _, p := range paths {
				rels = append(rels, relTo(root, p))
			}
			return xerrors.Wrap(errEgressDrift, "make egress-apply してコミットしてください: "+strings.Join(rels, ", "))
		}
		log.Printf("✅ 全ジョブの allowed-endpoints が %s 通りです", ssotFile)

		return nil
	}
	for _, f := range paths {
		if err := os.WriteFile(f, []byte(changes[f]), filePerm); err != nil { //nolint:gosec // path from cwd glob
			return xerrors.Wrap(err, "write "+relTo(root, f))
		}
		log.Printf("  updated %s", relTo(root, f))
	}
	log.Printf("✅ %d ファイルへ反映しました", len(paths))

	return nil
}

func relTo(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}
