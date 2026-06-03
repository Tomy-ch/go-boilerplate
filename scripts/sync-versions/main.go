// scripts/sync-versions は mise.toml [tools] の go / node / python を SSOT として、
// 以下のファイルへ伝播する Go 製の小さなツールです。
//
//   - go.mod                    `go X.Y.Z` directive
//   - docker/server/Dockerfile  `FROM golang:X.Y.Z-alpine`
//   - docker/tools/Dockerfile   `FROM golang:X.Y.Z-alpine`
//     `FROM node:X.Y-alpine`
//     `FROM python:X.Y.Z-slim`
//
// 設計:
//   - mise.toml は `[tools]` table 配下の `go` / `node` / `python` のみを参照する
//     行ベース parser を自前で持つ（外部依存ゼロ、production binary への影響なし）。
//   - 各 rule の事前 validate を全て通してから書き出すため partial state を残さない。
//   - 期待マッチ数（expectedCount）未満の rule が1つでもあれば、一切書かずに非ゼロ終了。
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
)

const (
	// filePerm は既存設定ファイルと同じ 0o644 で上書きする。
	filePerm fs.FileMode = 0o644
	// fromRegexpGroups は FROM タグ書き換え regex が持つ capture group 数 + 1（全体マッチ含む）。
	fromRegexpGroups = 3
	// 各ファイル内に出現する image 参照の期待件数（drift 検出の下限）。
	// 値は現実装の出現箇所数に合わせており、件数が下回ると abort する。
	serverDockerfileGolangCount = 3
	toolsDockerfileGolangCount  = 2
	dockerReadmeGolangCount     = 3
	serverReadmeGolangCount     = 2
)

var (
	miseSectionRe = regexp.MustCompile(`^\[([^\]]+)\]`)
	miseKeyRe     = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*=\s*"([^"]+)"`)
	goModRe       = regexp.MustCompile(`(?m)^go \d+(?:\.\d+){0,2}$`)
	// Dockerfile の `FROM ...` 行用。`(?m)^[^#]*?` で行頭にアンカーしつつ `#` で始まる
	// コメント行を除外する（コメントアウトされた FROM 行を誤って書き換えないため）。
	golangFromRe = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+golang:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	nodeFromRe   = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+node:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	pythonFromRe = regexp.MustCompile(`(?m)^[^#]*?(FROM\s+python:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	// docker/**/README.md の Markdown テーブル内 image 参照用。バッククォートで括られた
	// `golang:X.Y.Z-alpine` 等にマッチする。バッククォートを capture group に含めることで
	// 置換時にも保持する。
	golangImageRe = regexp.MustCompile("(`golang:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
	nodeImageRe   = regexp.MustCompile("(`node:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
	pythonImageRe = regexp.MustCompile("(`python:)" + `\d+(?:\.\d+){0,2}` + "(-[\\w.-]+`)")
)

// runtimeVersions は mise.toml [tools] から抽出した3ランタイムのバージョン文字列。
type runtimeVersions struct {
	Go     string
	Node   string
	Python string
}

// rule はあるファイルの正規表現マッチ箇所を一つの version で置換する単位。
type rule struct {
	label         string
	file          string
	re            *regexp.Regexp
	version       string
	replace       func(match string) string
	expectedCount int
}

// fileState は同一ファイルへの複数 rule を1回の書き出しにまとめるための中間状態。
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

// parseMiseTOML は mise.toml の [tools] table 配下の go/node/python キーだけを抽出する
// 用途特化の最小 parser。完全な TOML 仕様には準拠しない。
func parseMiseTOML(path string) (runtimeVersions, error) {
	var v runtimeVersions
	f, err := os.Open(path) //nolint:gosec // path is constructed from cwd + literal filename
	if err != nil {
		return v, fmt.Errorf("open: %w", err)
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
		if currentSection != "tools" {
			continue
		}
		m := miseKeyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch m[1] {
		case "go":
			v.Go = m[2]
		case "node":
			v.Node = m[2]
		case "python":
			v.Python = m[2]
		}
	}
	if err := scanner.Err(); err != nil {
		return v, fmt.Errorf("scan: %w", err)
	}
	return v, nil
}

func printSource(v runtimeVersions) {
	log.Println("Source: mise.toml")
	log.Printf("  go     = %s", emptyAs(v.Go, "(unset)"))
	log.Printf("  node   = %s", emptyAs(v.Node, "(unset)"))
	log.Printf("  python = %s", emptyAs(v.Python, "(unset)"))
}

// dockerfileRule は Dockerfile 内の `FROM <image>:X.Y.Z-<suffix>` 行に対する rule を構築する。
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

// readmeRule は docker/**/README.md 内のバッククォート括り `image:X.Y.Z-<suffix>` 参照に対する
// rule を構築する。
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
		// Dockerfile 内 FROM 行
		dockerfileRule("docker/server/Dockerfile",
			"docker/server/Dockerfile (golang base)", golangFromRe, v.Go, serverDockerfileGolangCount),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (golang base)", golangFromRe, v.Go, toolsDockerfileGolangCount),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (node base)", nodeFromRe, v.Node, 1),
		dockerfileRule("docker/tools/Dockerfile",
			"docker/tools/Dockerfile (python base)", pythonFromRe, v.Python, 1),
		// docker/**/README.md 内 image 参照（バッククォート括り）
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
	}
}

func fromReplacer(re *regexp.Regexp, version string) func(string) string {
	return func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < fromRegexpGroups {
			return match
		}
		return sub[1] + version + sub[2]
	}
}

// validateRules は version 未設定と file 不在を事前検査する。
func validateRules(rules []rule, root string) []string {
	var errs []string
	for _, r := range rules {
		if r.version == "" {
			errs = append(errs, fmt.Sprintf(
				"%s: mise.toml [tools] に対応する key が未設定", r.label))
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

// computeChanges は実書き出しを行わずに、各 file の最終内容を計算する。
// 期待マッチ数未満の rule があれば該当 rule のエラーを返す。
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

// writeChanges は事前 validate を通過した状態を file 単位で1回ずつ書き出す。
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
			return fmt.Errorf("write %s: %w", file, err)
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

func emptyAs(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
