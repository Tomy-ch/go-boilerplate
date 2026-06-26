// 不変条件: 全 rule の事前 validate を通してから初めて書き出す。期待マッチ数を
// 満たさない rule が1つでもあれば一切書かずに非ゼロ終了し、partial state を残さない。
package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go-boilerplate/pkg/xerrors"
)

const (
	filePerm fs.FileMode = 0o644
	// fromRegexpGroups は FROM タグ書き換え regex の capture group 数 + 1（全体マッチ含む）。
	fromRegexpGroups = 3
	// drift 検出の下限件数。現実装の出現箇所数に揃えており、下回ると abort する。
	serverDockerfileGolangCount = 3
	toolsDockerfileGolangCount  = 2
	dockerReadmeGolangCount     = 3
	serverReadmeGolangCount     = 2
)

var (
	miseSectionRe = regexp.MustCompile(`^\[([^\]]+)\]`)
	miseKeyRe     = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*=\s*"([^"]+)"`)
	goModRe       = regexp.MustCompile(`(?m)^go \d+(?:\.\d+){0,2}$`)
	// `(?m)^[^#]*?` でコメント行（先頭 `#`）を除外する。
	golangFromRe = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+golang:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	nodeFromRe   = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+node:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	pythonFromRe = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+python:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	// バッククォートを capture に含めることで置換後も保持する。
	golangImageRe    = regexp.MustCompile("(`golang:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
	nodeImageRe      = regexp.MustCompile("(`node:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
	pythonImageRe    = regexp.MustCompile("(`python:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
	miseDockerfileRe = regexp.MustCompile(`(MISE_VERSION=v)\d+(?:\.\d+){0,2}()`)
	miseActionRe     = regexp.MustCompile(`(?m)^([ \t]+version: )\d+(?:\.\d+){0,2}([ \t]*)$`)
	// docker-compose.yaml の otel-lgtm image タグ。suffix は空 capture でタグ末尾を保持する。
	otelLgtmImageRe = regexp.MustCompile("(grafana/otel-lgtm:)" + `\d+(?:\.\d+){0,2}` + "()")
)

// runtimeVersions は mise.toml から抽出したバージョン文字列。
type runtimeVersions struct {
	Go       string
	Node     string
	Python   string
	Mise     string
	OtelLgtm string
}

// rule はファイル内の regex マッチ箇所を 1 つの version で置換する単位。
type rule struct {
	label         string
	file          string
	re            *regexp.Regexp
	version       string
	replace       func(match string) string
	expectedCount int
}

// fileState は同一ファイルに複数 rule を順次適用する間の中間状態。
type fileState struct {
	original string
	current  string
	applied  []string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ getwd: %v", err)
	}

	v, err := parseMiseTOML(filepath.Join(root, "mise.toml"))
	if err != nil {
		log.Fatalf("❌ mise.toml のパースに失敗: %v", err)
	}

	printSource(v)

	rules := buildRules(v)

	if errs := validateRules(rules, root); len(errs) > 0 {
		reportAndExit("Validation errors（書き換えは行いません）", errs)
	}

	states, errs := computeChanges(rules, root)
	if len(errs) > 0 {
		reportAndExit("Match-count errors（書き換えは行いません）", errs)
	}

	if err := writeChanges(states, root); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

// parseMiseTOML は mise.toml の [tools] table 配下の go / node / python キーを抽出する。
// 完全な TOML 仕様には準拠しない用途特化の最小 parser。
func parseMiseTOML(path string) (runtimeVersions, error) {
	var v runtimeVersions
	f, err := os.Open(path) //nolint:gosec // path is constructed from cwd + literal filename
	if err != nil {
		return v, xerrors.Wrap(err, "open")
	}
	defer func() { _ = f.Close() }()

	currentSection := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := miseSectionRe.FindStringSubmatch(line); m != nil {
			currentSection = m[1]
			continue
		}
		if m := miseKeyRe.FindStringSubmatch(line); m != nil {
			applyMiseKV(&v, currentSection, m[1], m[2])
		}
	}
	if err := scanner.Err(); err != nil {
		return v, xerrors.Wrap(err, "scan")
	}
	return v, nil
}

// applyMiseKV は section / key / value を runtimeVersions の該当フィールドへ反映する。
// ルートの min_version、[env] の OTEL_LGTM_VERSION、[tools] の go/node/python のみを対象とする。
func applyMiseKV(v *runtimeVersions, section, key, val string) {
	switch section {
	case "":
		if key == "min_version" {
			v.Mise = val
		}
	case "env":
		if key == "OTEL_LGTM_VERSION" {
			v.OtelLgtm = val
		}
	case "tools":
		switch key {
		case "go":
			v.Go = val
		case "node":
			v.Node = val
		case "python":
			v.Python = val
		}
	}
}

func printSource(v runtimeVersions) {
	log.Println("Source: mise.toml")
	log.Printf("  go          = %s", emptyAs(v.Go))
	log.Printf("  node        = %s", emptyAs(v.Node))
	log.Printf("  python      = %s", emptyAs(v.Python))
	log.Printf("  min_version = %s", emptyAs(v.Mise))
	log.Printf("  otel-lgtm   = %s", emptyAs(v.OtelLgtm))
}

func dockerfileRule(file, label string, re *regexp.Regexp, version string, count int) rule {
	return rule{
		label:         label,
		file:          file,
		re:            re,
		version:       version,
		replace:       fromReplacer(re, version),
		expectedCount: count,
	}
}

func readmeRule(file, label string, re *regexp.Regexp, version string, count int) rule {
	return rule{
		label:         label,
		file:          file,
		re:            re,
		version:       version,
		replace:       fromReplacer(re, version),
		expectedCount: count,
	}
}

func buildRules(v runtimeVersions) []rule {
	return []rule{
		{
			label:         "go.mod (go directive)",
			file:          "go.mod",
			re:            goModRe,
			version:       v.Go,
			replace:       func(string) string { return "go " + v.Go },
			expectedCount: 1,
		},
		dockerfileRule("docker/server/Dockerfile",
			"docker/server/Dockerfile (golang base)", golangFromRe, v.Go, serverDockerfileGolangCount),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (golang base)", golangFromRe, v.Go, toolsDockerfileGolangCount),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (node base)", nodeFromRe, v.Node, 1),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (python base)", pythonFromRe, v.Python, 1),
		readmeRule("docker/README.md",
			"docker/README.md (golang image)", golangImageRe, v.Go, dockerReadmeGolangCount),
		readmeRule("docker/README.md",
			"docker/README.md (node image)", nodeImageRe, v.Node, 1),
		readmeRule("docker/README.md",
			"docker/README.md (python image)", pythonImageRe, v.Python, 1),
		readmeRule("docker/README.ja.md",
			"docker/README.ja.md (golang image)", golangImageRe, v.Go, dockerReadmeGolangCount),
		readmeRule("docker/README.ja.md",
			"docker/README.ja.md (node image)", nodeImageRe, v.Node, 1),
		readmeRule("docker/README.ja.md",
			"docker/README.ja.md (python image)", pythonImageRe, v.Python, 1),
		readmeRule("docker/server/README.md",
			"docker/server/README.md (golang image)", golangImageRe, v.Go, serverReadmeGolangCount),
		readmeRule("docker/server/README.ja.md",
			"docker/server/README.ja.md (golang image)", golangImageRe, v.Go, serverReadmeGolangCount),
		readmeRule("docker/tools/README.md",
			"docker/tools/README.md (golang image)", golangImageRe, v.Go, 1),
		readmeRule("docker/tools/README.md",
			"docker/tools/README.md (node image)", nodeImageRe, v.Node, 1),
		readmeRule("docker/tools/README.md",
			"docker/tools/README.md (python image)", pythonImageRe, v.Python, 1),
		readmeRule("docker/tools/README.ja.md",
			"docker/tools/README.ja.md (golang image)", golangImageRe, v.Go, 1),
		readmeRule("docker/tools/README.ja.md",
			"docker/tools/README.ja.md (node image)", nodeImageRe, v.Node, 1),
		readmeRule("docker/tools/README.ja.md",
			"docker/tools/README.ja.md (python image)", pythonImageRe, v.Python, 1),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (mise version)", miseDockerfileRe, v.Mise, 1),
		dockerfileRule("docker/server/Dockerfile",
			"docker/server/Dockerfile (mise version)", miseDockerfileRe, v.Mise, 1),
		dockerfileRule(".github/workflows/go-lint.yaml",
			"go-lint.yaml (mise-action version)", miseActionRe, v.Mise, 1),
		dockerfileRule(".github/workflows/gen-db-artifacts-check.yaml",
			"gen-db-artifacts-check.yaml (mise-action version)", miseActionRe, v.Mise, 1),
		dockerfileRule(".github/workflows/vulnerability-check.yaml",
			"vulnerability-check.yaml (mise-action version)", miseActionRe, v.Mise, 1),
		dockerfileRule("docker-compose.yaml",
			"docker-compose.yaml (otel-lgtm image)", otelLgtmImageRe, v.OtelLgtm, 1),
	}
}

// fromReplacer は 2 capture group regex の version 部分だけを置換し、prefix / suffix を保持する replace 関数を返す。
func fromReplacer(re *regexp.Regexp, version string) func(string) string {
	return func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < fromRegexpGroups {
			return match
		}
		return sub[1] + version + sub[2]
	}
}

// validateRules は version 未設定の rule とファイル不在の rule をエラーリストで返す。
func validateRules(rules []rule, root string) []string {
	var errs []string
	for _, r := range rules {
		if r.version == "" {
			errs = append(errs, r.label+": mise.toml に対応する version が未設定")
		}
	}
	for _, r := range rules {
		if _, err := os.Stat(filepath.Join(root, r.file)); err != nil {
			errs = append(errs, fmt.Sprintf(
				"%s: %s が見つかりません", r.label, r.file))
		}
	}
	return errs
}

// computeChanges は各 file の最終内容を計算する（書き出しは行わない）。期待マッチ数未満の rule があればエラーを返す。
func computeChanges(rules []rule, root string) (map[string]*fileState, []string) {
	states := map[string]*fileState{}
	var errs []string
	for _, r := range rules {
		st, ok := states[r.file]
		if !ok {
			data, err := os.ReadFile(filepath.Join(root, r.file)) //nolint:gosec // path from cwd + rule.file
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: read %s: %v", r.label, r.file, err))
				continue
			}
			st = &fileState{original: string(data), current: string(data)}
			states[r.file] = st
		}
		matches := r.re.FindAllStringIndex(st.current, -1)
		if len(matches) < r.expectedCount {
			errs = append(errs, fmt.Sprintf(
				"%s: マッチ件数 %d が期待値 %d 未満（regex: %s）",
				r.label, len(matches), r.expectedCount, r.re))
			continue
		}
		st.current = r.re.ReplaceAllStringFunc(st.current, r.replace)
		st.applied = append(st.applied, fmt.Sprintf("%s x%d", r.label, len(matches)))
	}
	return states, errs
}

// writeChanges は各 file を 1 回ずつ書き出す。変更が無いファイルは skip する。
func writeChanges(states map[string]*fileState, root string) error {
	anyChange := false
	for file, st := range states {
		if st.original == st.current {
			log.Printf("No change: %s", file)
			continue
		}
		if err := os.WriteFile(
			filepath.Join(root, file), []byte(st.current), filePerm,
		); err != nil {
			return xerrors.Wrap(err, "write "+file)
		}
		log.Printf("Updated: %s [%s]", file, strings.Join(st.applied, ", "))
		anyChange = true
	}
	if anyChange {
		log.Println("Done.")
	} else {
		log.Println("Done (no changes).")
	}
	return nil
}

func reportAndExit(title string, errs []string) {
	log.Println("")
	log.Printf("❌ %s:", title)
	for _, e := range errs {
		log.Printf("  - %s", e)
	}
	os.Exit(1)
}

func emptyAs(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
