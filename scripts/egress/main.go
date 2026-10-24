// Package main は bullfrog の通信制限ステップの入力を SSOT から生成・検証するツール。
//
//	apply: .github/egress.toml を元に各 workflow の bullfrog ステップの with ブロックを書き換える。
//	check: apply と同じ判定を書き換えなしで行い、ドリフトがあれば非ゼロ終了する（CI / hook 用）。
//
// 詳細は .github/workflows/README.md の Runner Hardening 節を参照。
package main

import (
	"fmt"
	"log"
	"net"
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
	// policyAudit は許可リストを持たない唯一の egress-policy。
	policyAudit = "audit"
	policyBlock = "block"
	// withIndentStep は with キーに対する入力キーの相対字下げ。一覧の値はさらに同じだけ深い。
	withIndentStep = 2
)

var (
	sectionRe   = regexp.MustCompile(`^\[(class|job|ip)\.(?:"([^"]+)"|([A-Za-z0-9_-]+))\]$`)
	scalarRe    = regexp.MustCompile(`^(\w+)\s*=\s*"([^"]*)"$`)
	arrayHeadRe = regexp.MustCompile(`^(\w+)\s*=\s*\[(.*)$`)
	quotedRe    = regexp.MustCompile(`"([^"]*)"`)
	jobIDRe     = regexp.MustCompile(`^  ([A-Za-z_][A-Za-z0-9_-]*):\s*(#.*)?$`)
	stepsKeyRe  = regexp.MustCompile(`^    steps:\s*$`)
	stepItemRe  = regexp.MustCompile(`^      - `)
	stepUsesRe  = regexp.MustCompile(`^(\s*)(- )?uses: bullfrogsec/bullfrog@[0-9a-f]{40} # v\S+\s*$`)
	policyRe    = regexp.MustCompile(`^\s*egress-policy:\s*(\S+)\s*$`)
	ipRe        = regexp.MustCompile(`^[0-9./]+$`)
	stepKeyRe   = regexp.MustCompile(`^( *)(- )?([A-Za-z_][A-Za-z0-9_-]*):`)
	hostRe      = regexp.MustCompile(`^(\*\.)?[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*$`)
	// actionRefRe は、生成器が所有する形でなくても拾うべき通信制限ステップへの言及。
	actionRefRe = regexp.MustCompile(`bullfrogsec/bullfrog|step-security/harden-runner`)
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
	// errSSOTHost は、ホストがポート付き・空白入り・途中にワイルドカードを持つ場合のエラー。
	errSSOTHost = xerrors.New("SSOT のホストがドメイン名として不正です")
	// errSSOTIP は、IP セクションの名前が IPv4 のアドレスでも CIDR でもない場合のエラー。
	errSSOTIP = xerrors.New("SSOT の IP が不正です")
	// errSSOTReason は、IP の reason やジョブの sudo の根拠が空の場合のエラー。
	errSSOTReason = xerrors.New("SSOT に根拠の無い宣言があります")
	// errWorkflowJobless は、どのジョブにも属さない通信制限ステップを検出した場合のエラー。
	errWorkflowJobless = xerrors.New("ジョブに属さない通信制限ステップがあります")
	// errStepMalformed は、通信制限ステップが生成器の所有する形でない場合のエラー。
	errStepMalformed = xerrors.New("通信制限ステップが想定外の形です")
	// errStepMissing は、ステップを持つジョブに通信制限ステップが無い場合のエラー。
	errStepMissing = xerrors.New("通信制限ステップを持たないジョブがあります")
	// errStepDuplicate は、1 ジョブに通信制限ステップが 2 つ以上ある場合のエラー。
	errStepDuplicate = xerrors.New("1 ジョブに通信制限ステップが複数あります")
	// errStepNotFirst は、通信制限ステップがジョブの先頭のステップでない場合のエラー。
	errStepNotFirst = xerrors.New("通信制限ステップがジョブの先頭にありません")
	// errPolicyMismatch は、workflow の egress-policy と SSOT の宣言が食い違う場合のエラー。
	errPolicyMismatch = xerrors.New("workflow の egress-policy と SSOT の宣言が食い違います")
	// errJobMissing は、workflow のジョブが SSOT に未登録の場合のエラー。
	errJobMissing = xerrors.New("SSOT に未登録のジョブがあります")
	// errJobOrphan は、SSOT にあるがどの workflow にも存在しないジョブがあった場合のエラー。
	errJobOrphan = xerrors.New("SSOT に対応する workflow が無いジョブがあります")
	// errEgressDrift は、check で SSOT との差分を検出した場合のエラー。
	errEgressDrift = xerrors.New("通信制限ステップの入力が SSOT からずれています")
)

// class は能力クラス 1 件。includes は継承元クラス、hosts は自クラスが足すドメイン。
type class struct {
	includes []string
	hosts    []string
}

// jobSpec は 1 ジョブの宣言。key は `<workflow ファイル名>:<job id>`。
// sudo は sudo を残す根拠で、空なら sudo を外す。
type jobSpec struct {
	classes []string
	extra   []string
	policy  string
	sudo    string
}

// ssot は .github/egress.toml の内容。order は宣言順（出力の並びを SSOT が決める）。
// ips は全ジョブが許す IP で、reasons はその根拠。
type ssot struct {
	classes   map[string]*class
	jobs      map[string]*jobSpec
	jobOrder  []string
	baseHosts []string
	ips       []string
	reasons   map[string]string
}

// orderedHosts は初出順を保ったまま重複を落とすホスト集合。
type orderedHosts struct {
	list []string
	seen map[string]bool
}

// step は workflow 内の通信制限ステップ 1 件の位置と中身。
type step struct {
	jobKey string
	// withLine は `with:` の行番号（0 起点）。
	withLine int
	// endLine は with ブロックの終端（0 起点・排他）。
	endLine int
	indent  int
	policy  string
}

// main は run を呼ぶだけの入口。判断は run に置く（scripts/README.md の Test Strategy）。
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

// parseSSOT は SSOT の本文を IP・クラス定義・ジョブ宣言へ分解する。
func parseSSOT(data string) (*ssot, error) {
	s := &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}, reasons: map[string]string{}}
	var curClass *class
	var curJob *jobSpec
	curIP := ""
	var seen map[string]bool

	lines := strings.Split(data, "\n")
	for i := 0; i < len(lines); i++ {
		stripped, err := stripValueComment(lines[i], i)
		if err != nil {
			return nil, err
		}
		line := strings.TrimSpace(stripped)
		if line == "" {
			continue
		}
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			name := m[2] + m[3]
			if err := s.openSection(m[1], name); err != nil {
				return nil, err
			}
			curClass, curJob, curIP = s.classes[name], s.jobs[name], ""
			if m[1] == "ip" {
				curIP = name
			}
			seen = map[string]bool{}
			continue
		}
		if curClass == nil && curJob == nil && curIP == "" {
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
		if curIP != "" {
			err = s.assignIP(curIP, key, values)
		} else {
			err = assign(curClass, curJob, key, values)
		}
		if err != nil {
			return nil, xerrors.Wrap(err, fmt.Sprintf("%d 行目", i+1))
		}
	}
	return s, s.validate()
}

// stripValueComment は SSOT の 1 行から、クオートの外の `#` 以降を落とす。
// クオートの中の `#` は値を途中で切ってしまうので、構文エラーとして返す。n は 0 起点の行番号。
func stripValueComment(line string, n int) (string, error) {
	inQuote := false
	for i, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == '#' && inQuote:
			return "", xerrors.Wrap(errSSOTSyntax, fmt.Sprintf("%d 行目: 値の中に # は書けません", n+1))
		case r == '#':
			return line[:i], nil
		}
	}
	return line, nil
}

// stripComment は workflow の行の行末コメントを落とす。YAML の値の中の `#` までは区別しない。
func stripComment(line string) string {
	if before, _, ok := strings.Cut(line, "#"); ok {
		return before
	}
	return line
}

// openSection は新しい `[class.x]` / `[job."x"]` / `[ip."x"]` を登録する。既出なら重複エラー。
func (s *ssot) openSection(kind, name string) error {
	if _, dup := s.classes[name]; dup && kind == "class" {
		return xerrors.Wrap(errSSOTDuplicate, "class "+name)
	}
	if _, dup := s.jobs[name]; dup && kind == "job" {
		return xerrors.Wrap(errSSOTDuplicate, "job "+name)
	}
	switch kind {
	case "class":
		s.classes[name] = &class{}
	case "ip":
		if slices.Contains(s.ips, name) {
			return xerrors.Wrap(errSSOTDuplicate, "ip "+name)
		}
		if err := validateIP(name); err != nil {
			return err
		}
		s.ips = append(s.ips, name)
	default:
		s.jobs[name] = &jobSpec{}
		s.jobOrder = append(s.jobOrder, name)
	}
	return nil
}

// parseAssignment は lines[i] から始まる代入を読み、キー・値・消費した最終行を返す。
// 配列は `]` が現れるまで後続行を取り込む。
func parseAssignment(lines []string, i int) (string, []string, int, error) {
	head, err := stripValueComment(lines[i], i)
	if err != nil {
		return "", nil, 0, err
	}
	line := strings.TrimSpace(head)
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
		if rest, err = stripValueComment(lines[i], i); err != nil {
			return "", nil, 0, err
		}
	}
}

// assign は読み取ったキーを、開いているクラスかジョブのセクションのフィールドへ載せる。
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
	case j != nil && key == "sudo" && len(values) == 1:
		if strings.TrimSpace(values[0]) == "" {
			return xerrors.Wrap(errSSOTReason, "sudo")
		}
		j.sudo = values[0]
	default:
		return xerrors.Wrap(errSSOTSyntax, "未知のキー "+key)
	}
	return nil
}

// assignIP は IP セクションのキーを載せる。持てるのは許す根拠の reason だけ。
func (s *ssot) assignIP(ip, key string, values []string) error {
	if key != "reason" || len(values) != 1 {
		return xerrors.Wrap(errSSOTSyntax, "未知のキー "+key)
	}
	if strings.TrimSpace(values[0]) == "" {
		return xerrors.Wrap(errSSOTReason, "ip "+ip)
	}
	s.reasons[ip] = values[0]
	return nil
}

// requireUnique は同一リスト内の重複を弾く。
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

// validateHost はホストが bullfrog の allowed-domains に書けるドメイン名であることを確かめる。
// ワイルドカードは先頭の `*.` だけを許す。bullfrog は英数字・`.`・`-`・`*` 以外を含む値を拒否して
// ステップごと失敗するので、`_` も書けない。
func validateHost(host string) error {
	if !hostRe.MatchString(host) {
		return xerrors.Wrap(errSSOTHost, host)
	}
	return nil
}

// validateIP は IP セクションの名前が IPv4 のアドレスか CIDR であることを確かめる。
// bullfrog は数字・`.`・`/` 以外を含む allowed-ips を拒否してステップごと失敗するので、IPv6 も書けない。
func validateIP(ip string) error {
	if !ipRe.MatchString(ip) {
		return xerrors.Wrap(errSSOTIP, ip)
	}
	if net.ParseIP(ip) != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(ip); err == nil {
		return nil
	}
	return xerrors.Wrap(errSSOTIP, ip)
}

// validate は SSOT 単体で判定できる不整合を返す。
func (s *ssot) validate() error {
	if base, ok := s.classes["base"]; ok {
		s.baseHosts = base.hosts
	}
	for _, ip := range s.ips {
		if _, ok := s.reasons[ip]; !ok {
			return xerrors.Wrap(errSSOTReason, "ip "+ip)
		}
	}
	for name, c := range s.classes {
		for _, inc := range c.includes {
			if _, ok := s.classes[inc]; !ok {
				return xerrors.Wrap(errSSOTUnknownClass, name+" includes "+inc)
			}
		}
		for _, h := range c.hosts {
			if err := validateHost(h); err != nil {
				return xerrors.Wrap(err, "class "+name)
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
			return xerrors.Wrap(errSSOTPolicy, key+": audit は許可リストを持ちません")
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
	for _, h := range j.extra {
		if err := validateHost(h); err != nil {
			return xerrors.Wrap(err, key)
		}
	}
	_, err := s.hostsFor(key)
	return err
}

// hostsFor は 1 ジョブの許可リストを算出する。base → classes（includes 展開込み）→ extra の順で、
// 重複は初出を残す。並びは SSOT の宣言順に固定される。
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

// scanWorkflow は 1 ファイルの通信制限ステップを読み、ジョブごとに 1 件ずつ返す。
// name は workflow のファイル名。ステップを持つジョブは、その先頭に通信制限ステップを持たなければならない。
func scanWorkflow(name string, lines []string) ([]step, error) {
	jobs := workflowJobs(lines)
	stepped := jobsWithSteps(lines, jobs)
	var steps []step
	found := map[string]bool{}

	for i := 0; i < len(lines); i++ {
		m := stepUsesRe.FindStringSubmatch(lines[i])
		if m == nil {
			if actionRefRe.MatchString(stripComment(lines[i])) {
				return nil, xerrors.Wrap(errStepMalformed, fmt.Sprintf("%s %d 行目: %q", name, i+1, strings.TrimSpace(lines[i])))
			}
			continue
		}
		if jobs[i] == "" {
			return nil, xerrors.Wrap(errWorkflowJobless, fmt.Sprintf("%s %d 行目", name, i+1))
		}
		key := name + ":" + jobs[i]
		if found[key] {
			return nil, xerrors.Wrap(errStepDuplicate, key)
		}
		found[key] = true
		if first := stepped[jobs[i]]; !withinFirstStep(lines, first, i) {
			return nil, xerrors.Wrap(errStepNotFirst, key)
		}
		indent := len(m[1]) + len(m[2])
		st, err := readStep(name, lines, i+1, indent)
		if err != nil {
			return nil, err
		}
		if err := checkStepKeys(lines, stepped[jobs[i]], st.endLine, indent); err != nil {
			return nil, xerrors.Wrap(err, key)
		}
		st.jobKey = key
		steps = append(steps, st)
		i = st.endLine - 1
	}
	for job := range stepped {
		if !found[name+":"+job] {
			return nil, xerrors.Wrap(errStepMissing, name+":"+job)
		}
	}
	return steps, nil
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

// jobsWithSteps は `steps:` を持つ job id と、その先頭のステップの行番号を返す。
// 再利用 workflow を呼ぶだけのジョブはステップを持たないので含まれない。
func jobsWithSteps(lines, jobs []string) map[string]int {
	out := map[string]int{}
	for i, line := range lines {
		if !stepsKeyRe.MatchString(line) || jobs[i] == "" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if stepItemRe.MatchString(lines[j]) {
				out[jobs[i]] = j
				break
			}
		}
	}
	return out
}

// withinFirstStep は usesLine が first から始まるステップの中にあるかを返す。
func withinFirstStep(lines []string, first, usesLine int) bool {
	if usesLine < first {
		return false
	}
	for j := first + 1; j <= usesLine; j++ {
		if stepItemRe.MatchString(lines[j]) {
			return false
		}
	}
	return true
}

// stepKeys は通信制限ステップが持ってよいキー。if や continue-on-error はステップを実行しないまま
// 検査を通すので持たせない。
var stepKeys = map[string]bool{"name": true, "id": true, "uses": true, "with": true}

// checkStepKeys は first から始まるステップのうち、with ブロックの後ろ（end 以降）と前にある
// ステップ直下のキーが stepKeys に収まることを確かめる。indent はステップのキーの字下げ。
func checkStepKeys(lines []string, first, end, indent int) error {
	for j := first; j < len(lines); j++ {
		if j > first && stepItemRe.MatchString(lines[j]) {
			break
		}
		line := lines[j]
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if j >= end && len(line)-len(strings.TrimLeft(line, " ")) < indent {
			break
		}
		m := stepKeyRe.FindStringSubmatch(line)
		if m == nil || len(m[1])+len(m[2]) != indent {
			continue
		}
		if !stepKeys[m[3]] {
			return xerrors.Wrap(errStepMalformed, fmt.Sprintf("%d 行目: 通信制限ステップに %s は書けません", j+1, m[3]))
		}
	}
	return nil
}

// readStep は uses の次の行から始まる with ブロックを読む。indent は uses キーの字下げ。
// ブロックの中身は生成器が所有するので、egress-policy 以外の行は読み飛ばして差し替える。
func readStep(name string, lines []string, withLine, indent int) (step, error) {
	pad := strings.Repeat(" ", indent)
	if withLine >= len(lines) || lines[withLine] != pad+"with:" {
		return step{}, xerrors.Wrap(errStepMalformed, fmt.Sprintf("%s %d 行目: uses の次に with がありません", name, withLine+1))
	}
	st := step{withLine: withLine, endLine: withLine + 1, indent: indent}
	inner := pad + strings.Repeat(" ", withIndentStep)
	for j := withLine + 1; j < len(lines) && strings.HasPrefix(lines[j], inner); j++ {
		if m := policyRe.FindStringSubmatch(lines[j]); m != nil && st.policy == "" {
			st.policy = m[1]
		}
		st.endLine = j + 1
	}
	if blockResumes(lines, st.endLine, inner) {
		return step{}, xerrors.Wrap(errStepMalformed, fmt.Sprintf("%s %d 行目: with ブロックの途中に空行かコメントがあります", name, st.endLine+1))
	}
	if st.policy == "" {
		return step{}, xerrors.Wrap(errStepMalformed, fmt.Sprintf("%s %d 行目: egress-policy がありません", name, withLine+1))
	}
	return st, nil
}

// blockResumes は from から続く空行とコメント行の後ろに、inner の字下げの行が再び現れるかを返す。
// 現れるなら、with ブロックが途中で切れて残りが書き換えから漏れる。
func blockResumes(lines []string, from int, inner string) bool {
	for j := from; j < len(lines); j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return j > from && strings.HasPrefix(lines[j], inner)
	}
	return false
}

// renderStep は 1 ジョブの with ブロックを、uses キーと同じ字下げから書き出す。
// 一覧は `|` で書く。bullfrog は入力を改行で割るので、`>` で畳むと 1 行の不正な値になる。
// audit は名前解決も絞らないので dns-policy と allowed-domains を書かない。enable-sudo は action の
// 既定が true なので常に明示する。
func renderStep(j *jobSpec, ips, hosts []string, indent int) []string {
	pad := strings.Repeat(" ", indent)
	in := pad + strings.Repeat(" ", withIndentStep)
	item := in + strings.Repeat(" ", withIndentStep)
	out := []string{pad + "with:", in + "egress-policy: " + j.policy}
	if j.policy == policyBlock {
		out = append(out, in+"dns-policy: allowed-domains-only")
	}
	out = append(out, fmt.Sprintf("%senable-sudo: %t", in, j.sudo != ""))
	if len(ips) > 0 {
		out = append(out, in+"allowed-ips: |")
		for _, ip := range ips {
			out = append(out, item+ip)
		}
	}
	if j.policy == policyBlock {
		out = append(out, in+"allowed-domains: |")
		for _, h := range hosts {
			out = append(out, item+h)
		}
	}
	return out
}

// applyOrCheck は SSOT を正本に通信制限ステップを書き換える。dryRun=true は書き込まず、差分があれば非ゼロ終了で報告する。
func applyOrCheck(root string, dryRun bool) error {
	raw, err := os.ReadFile(filepath.Join(root, ssotFile)) //nolint:gosec // path is cwd + literal filename
	if err != nil {
		return xerrors.Wrap(err, "read "+ssotFile)
	}
	s, err := parseSSOT(string(raw))
	if err != nil {
		return err
	}
	var files []string
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matched, err := filepath.Glob(filepath.Join(root, ".github", "workflows", pattern))
		if err != nil {
			return xerrors.Wrap(err, "glob workflows")
		}
		files = append(files, matched...)
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
// 書き出す前に全て確定させる方針は scripts/README.md の Test Strategy が持つ。
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

// rewriteFile は 1 ファイルの通信制限ステップの with ブロックを SSOT 由来の内容へ差し替えた本文を返す。
func (s *ssot) rewriteFile(name, data string, used map[string]bool) (string, error) {
	lines := strings.Split(data, "\n")
	steps, err := scanWorkflow(name, lines)
	if err != nil {
		return "", err
	}
	for _, st := range steps {
		if err := s.checkPolicy(st.jobKey, st.policy); err != nil {
			return "", err
		}
		used[st.jobKey] = true
	}
	// 後ろから差し替えて、先行ステップの行番号を保つ。
	for _, v := range slices.Backward(steps) {
		st := v
		hosts, err := s.hostsFor(st.jobKey)
		if err != nil {
			return "", err
		}
		rendered := renderStep(s.jobs[st.jobKey], s.ips, hosts, st.indent)
		lines = append(lines[:st.withLine:st.withLine], append(rendered, lines[st.endLine:]...)...)
	}
	return strings.Join(lines, "\n"), nil
}

// checkPolicy は workflow 側の egress-policy が SSOT の宣言と揃うことを検証する。
// audit と block の切り替えは許可リストの有無を変えるので、生成で黙って書き換えず人が両方を直す。
func (s *ssot) checkPolicy(key, policy string) error {
	j, ok := s.jobs[key]
	if !ok {
		return xerrors.Wrap(errJobMissing, key+"（"+ssotFile+" へ追加してください）")
	}
	if j.policy != policy {
		return xerrors.Wrap(errPolicyMismatch, fmt.Sprintf("%s: workflow=%s SSOT=%s", key, policy, j.policy))
	}
	return nil
}

// orphanJobs は SSOT にあるがどの workflow にも現れないジョブキーを返す。
// これをエラーにするのは fail-closed の一環（scripts/README.md の egress 行）。
// sudo の宣言はジョブのセクションの中にあるので、ジョブと一緒に孤児として捕まる。
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

// writeChanges は planFiles が確定した書き換えを反映する。dryRun の扱いは applyOrCheck を参照。
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
		log.Printf("✅ 全ジョブの通信制限ステップが %s 通りです", ssotFile)

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
